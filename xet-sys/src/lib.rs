//! xet-sys — Rust C API static library for the Go xet package.
//!
//! This crate exposes the five functions declared in `xet/xet.h` as a
//! C-compatible `staticlib` that the Go CGo layer links against.  Each
//! function wraps the corresponding asynchronous xet-core operation with a
//! lazily-initialised Tokio runtime, so callers do not need to manage an
//! async context.
//!
//! # Memory ownership
//!
//! Every `*mut Xet*Result` returned to Go is heap-allocated via `Box::into_raw`.
//! All embedded C strings are heap-allocated via `CString::into_raw`.
//! Go must release result structs by calling the matching `xet_free_*` function.
#![allow(clippy::missing_safety_doc)]

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};
use std::ptr;
use std::sync::OnceLock;

use xet_pkg::legacy::{Sha256Policy, XetFileInfo, data_client};
use xet_data::deduplication::{Chunker, constants::TARGET_CHUNK_SIZE};
use xet_core_structures::merklehash::{MerkleHash, compute_data_hash, file_hash, xorb_hash};
use xet_core_structures::metadata_shard::chunk_verification::range_hash_from_chunks;
use xet_core_structures::xorb_object::deserialize_chunks;

// ---------------------------------------------------------------------------
// C struct mirrors — must match xet/xet.h exactly (#[repr(C)])
// ---------------------------------------------------------------------------

#[repr(C)]
pub struct XetTokenInfo {
    pub token: *const c_char,
    pub expiry: u64,
}

#[repr(C)]
pub struct XetUploadInfo {
    pub hash: *mut c_char,
    pub file_size: u64,
    pub sha256: *mut c_char,
}

#[repr(C)]
pub struct XetUploadResult {
    pub items: *mut XetUploadInfo,
    pub count: libc::size_t,
    pub error: *mut c_char,
}

#[repr(C)]
pub struct XetDownloadInfo {
    pub destination_path: *const c_char,
    pub hash: *const c_char,
    pub file_size: i64,
}

#[repr(C)]
pub struct XetDownloadResult {
    pub paths: *mut *mut c_char,
    pub count: libc::size_t,
    pub error: *mut c_char,
}

// ---------------------------------------------------------------------------
// Tokio runtime — created once, reused for every FFI call
// ---------------------------------------------------------------------------

fn runtime() -> &'static tokio::runtime::Runtime {
    static RT: OnceLock<tokio::runtime::Runtime> = OnceLock::new();
    RT.get_or_init(|| {
        tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()
            .expect("xet-sys: failed to create Tokio runtime")
    })
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

/// Convert a nullable C string pointer to an owned `String`.
unsafe fn c_str_opt(ptr: *const c_char) -> Option<String> {
    if ptr.is_null() {
        None
    } else {
        Some(CStr::from_ptr(ptr).to_string_lossy().into_owned())
    }
}

/// Convert an `Option<&str>` to a heap-allocated C string (or null).
fn opt_str_to_c(s: Option<&str>) -> *mut c_char {
    match s {
        Some(s) => CString::new(s)
            .map(|c| c.into_raw())
            .unwrap_or(ptr::null_mut()),
        None => ptr::null_mut(),
    }
}

/// Read the token / expiry from a nullable `XetTokenInfo` pointer.
unsafe fn parse_token(ti: *const XetTokenInfo) -> Option<(String, u64)> {
    if ti.is_null() {
        return None;
    }
    let t = &*ti;
    c_str_opt(t.token).map(|tok| (tok, t.expiry))
}

/// Build an error-only `XetUploadResult`.
fn upload_err(msg: &str) -> *mut XetUploadResult {
    Box::into_raw(Box::new(XetUploadResult {
        items: ptr::null_mut(),
        count: 0,
        error: opt_str_to_c(Some(msg)),
    }))
}

/// Build an error-only `XetDownloadResult`.
fn download_err(msg: &str) -> *mut XetDownloadResult {
    Box::into_raw(Box::new(XetDownloadResult {
        paths: ptr::null_mut(),
        count: 0,
        error: opt_str_to_c(Some(msg)),
    }))
}

/// Convert a `Vec<XetFileInfo>` into a heap-allocated `XetUploadResult`.
fn upload_infos_to_c(infos: Vec<XetFileInfo>) -> *mut XetUploadResult {
    let count = infos.len();
    let mut items: Vec<XetUploadInfo> = infos
        .iter()
        .map(|fi| XetUploadInfo {
            hash: opt_str_to_c(Some(fi.hash())),
            file_size: fi.file_size().unwrap_or(0),
            sha256: opt_str_to_c(fi.sha256()),
        })
        .collect();

    let items_ptr = items.as_mut_ptr();
    std::mem::forget(items); // transfer ownership to C caller

    Box::into_raw(Box::new(XetUploadResult {
        items: items_ptr,
        count,
        error: ptr::null_mut(),
    }))
}

/// Convert a `Vec<String>` of destination paths into a heap-allocated `XetDownloadResult`.
fn download_paths_to_c(paths: Vec<String>) -> *mut XetDownloadResult {
    let count = paths.len();
    let mut cptrs: Vec<*mut c_char> = paths
        .iter()
        .map(|p| opt_str_to_c(Some(p.as_str())))
        .collect();

    let paths_ptr = cptrs.as_mut_ptr();
    std::mem::forget(cptrs); // transfer ownership to C caller

    Box::into_raw(Box::new(XetDownloadResult {
        paths: paths_ptr,
        count,
        error: ptr::null_mut(),
    }))
}

// ---------------------------------------------------------------------------
// Public C API
// ---------------------------------------------------------------------------

/// Upload local files to HuggingFace Xet storage.
///
/// # Safety
/// All pointer arguments must be valid for the duration of the call.
/// The returned pointer must be freed with `xet_free_upload_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_upload_files(
    file_paths: *const *const c_char,
    file_count: libc::size_t,
    endpoint: *const c_char,
    token_info: *const XetTokenInfo,
    sha256s: *const *const c_char,
    sha256_count: libc::size_t,
    skip_sha256: c_int,
) -> *mut XetUploadResult {
    let paths: Vec<String> = (0..file_count)
        .filter_map(|i| c_str_opt(*file_paths.add(i)))
        .collect();

    let sha256_policies: Vec<Sha256Policy> = if skip_sha256 != 0 {
        vec![Sha256Policy::Skip; file_count]
    } else if !sha256s.is_null() && sha256_count == file_count {
        (0..sha256_count)
            .map(|i| match c_str_opt(*sha256s.add(i)) {
                Some(s) => Sha256Policy::from_hex(&s),
                None => Sha256Policy::Compute,
            })
            .collect()
    } else {
        vec![Sha256Policy::Compute; file_count]
    };

    let ep = c_str_opt(endpoint);
    let token = parse_token(token_info);

    match runtime().block_on(data_client::upload_async(
        paths,
        sha256_policies,
        ep,
        token,
        None, // token_refresher
        None, // progress_updater
        None, // custom_headers
    )) {
        Ok(infos) => upload_infos_to_c(infos),
        Err(e) => upload_err(&e.to_string()),
    }
}

/// Compute Xet content-hashes for local files without uploading.
///
/// # Safety
/// All pointer arguments must be valid for the duration of the call.
/// The returned pointer must be freed with `xet_free_upload_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_hash_files(
    file_paths: *const *const c_char,
    file_count: libc::size_t,
) -> *mut XetUploadResult {
    let paths: Vec<String> = (0..file_count)
        .filter_map(|i| c_str_opt(*file_paths.add(i)))
        .collect();

    match runtime().block_on(data_client::hash_files_async(paths)) {
        Ok(infos) => upload_infos_to_c(infos),
        Err(e) => upload_err(&e.to_string()),
    }
}

/// Download files from HuggingFace Xet storage to the local filesystem.
///
/// # Safety
/// All pointer arguments must be valid for the duration of the call.
/// The returned pointer must be freed with `xet_free_download_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_download_files(
    files: *const XetDownloadInfo,
    file_count: libc::size_t,
    endpoint: *const c_char,
    token_info: *const XetTokenInfo,
) -> *mut XetDownloadResult {
    let file_infos: Vec<(XetFileInfo, String)> = (0..file_count)
        .filter_map(|i| {
            let d = &*files.add(i);
            let hash = c_str_opt(d.hash)?;
            let dest = c_str_opt(d.destination_path)?;
            let xfi = if d.file_size >= 0 {
                XetFileInfo::new(hash, d.file_size as u64)
            } else {
                XetFileInfo::new_hash_only(hash)
            };
            Some((xfi, dest))
        })
        .collect();

    let ep = c_str_opt(endpoint);
    let token = parse_token(token_info);

    match runtime().block_on(data_client::download_async(
        file_infos,
        ep,
        token,
        None, // token_refresher
        None, // progress_updaters
        None, // custom_headers
    )) {
        Ok(paths) => download_paths_to_c(paths),
        Err(e) => download_err(&e.to_string()),
    }
}

/// Release a `XetUploadResult` returned by `xet_upload_files` or `xet_hash_files`.
///
/// Passing `NULL` is a no-op.
///
/// # Safety
/// `result` must have been returned by this library and not previously freed.
#[no_mangle]
pub unsafe extern "C" fn xet_free_upload_result(result: *mut XetUploadResult) {
    if result.is_null() {
        return;
    }
    let r = Box::from_raw(result);
    if !r.items.is_null() {
        // Reconstruct the Vec so Rust will drop the memory correctly.
        let items = Vec::from_raw_parts(r.items, r.count, r.count);
        for item in items {
            if !item.hash.is_null() {
                drop(CString::from_raw(item.hash));
            }
            if !item.sha256.is_null() {
                drop(CString::from_raw(item.sha256));
            }
        }
    }
    if !r.error.is_null() {
        drop(CString::from_raw(r.error));
    }
    // `r` (the Box) is dropped here, freeing the XetUploadResult itself.
}

/// Release a `XetDownloadResult` returned by `xet_download_files`.
///
/// Passing `NULL` is a no-op.
///
/// # Safety
/// `result` must have been returned by this library and not previously freed.
#[no_mangle]
pub unsafe extern "C" fn xet_free_download_result(result: *mut XetDownloadResult) {
    if result.is_null() {
        return;
    }
    let r = Box::from_raw(result);
    if !r.paths.is_null() {
        let ptrs = Vec::from_raw_parts(r.paths, r.count, r.count);
        for ptr in ptrs {
            if !ptr.is_null() {
                drop(CString::from_raw(ptr));
            }
        }
    }
    if !r.error.is_null() {
        drop(CString::from_raw(r.error));
    }
    // `r` (the Box) is dropped here, freeing the XetDownloadResult itself.
}

// ---------------------------------------------------------------------------
// C struct mirrors for chunk / hash / xorb-check — must match xet/xet.h
// ---------------------------------------------------------------------------

#[repr(C)]
pub struct XetChunkInfo {
    pub hash: *mut c_char,
    pub size: u64,
}

#[repr(C)]
pub struct XetChunkResult {
    pub items: *mut XetChunkInfo,
    pub count: libc::size_t,
    pub error: *mut c_char,
}

#[repr(C)]
pub struct XetHashResult {
    pub hash: *mut c_char,
    pub error: *mut c_char,
}

#[repr(C)]
pub struct XetXorbCheckResult {
    pub xorb_hash: *mut c_char,
    pub chunks: *mut XetChunkInfo,
    pub chunk_count: libc::size_t,
    pub total_bytes: u64,
    pub error: *mut c_char,
}

// ---------------------------------------------------------------------------
// Internal helpers for new result types
// ---------------------------------------------------------------------------

fn chunk_err(msg: &str) -> *mut XetChunkResult {
    Box::into_raw(Box::new(XetChunkResult {
        items: ptr::null_mut(),
        count: 0,
        error: opt_str_to_c(Some(msg)),
    }))
}

fn hash_err(msg: &str) -> *mut XetHashResult {
    Box::into_raw(Box::new(XetHashResult {
        hash: ptr::null_mut(),
        error: opt_str_to_c(Some(msg)),
    }))
}

fn xorb_check_err(msg: &str) -> *mut XetXorbCheckResult {
    Box::into_raw(Box::new(XetXorbCheckResult {
        xorb_hash: ptr::null_mut(),
        chunks: ptr::null_mut(),
        chunk_count: 0,
        total_bytes: 0,
        error: opt_str_to_c(Some(msg)),
    }))
}

/// Convert a `Vec<(MerkleHash, u64)>` (chunk_hash, chunk_size) pairs into a
/// heap-allocated `XetChunkResult`.
fn chunk_pairs_to_c(pairs: Vec<(MerkleHash, u64)>) -> *mut XetChunkResult {
    let count = pairs.len();
    let mut items: Vec<XetChunkInfo> = pairs
        .iter()
        .map(|(hash, size)| XetChunkInfo {
            hash: opt_str_to_c(Some(&hash.hex())),
            size: *size,
        })
        .collect();

    let items_ptr = items.as_mut_ptr();
    std::mem::forget(items);

    Box::into_raw(Box::new(XetChunkResult {
        items: items_ptr,
        count,
        error: ptr::null_mut(),
    }))
}

/// Chunk a byte slice using the xet content-defined chunker; returns pairs of
/// (chunk_hash, chunk_size).
fn do_chunk_bytes(data: &[u8]) -> Vec<(MerkleHash, u64)> {
    let mut chunker = Chunker::new(*TARGET_CHUNK_SIZE);
    let mut pairs = Vec::new();
    let chunks = chunker.next_block(data, false);
    for chunk in chunks {
        let size = chunk.data.len() as u64;
        pairs.push((chunk.hash, size));
    }
    if let Some(chunk) = chunker.finish() {
        let size = chunk.data.len() as u64;
        pairs.push((chunk.hash, size));
    }
    pairs
}

/// Parse an array of C string pointers into a `Vec<String>`.
unsafe fn c_strings_to_vec(ptrs: *const *const c_char, count: libc::size_t) -> Vec<String> {
    (0..count)
        .filter_map(|i| c_str_opt(*ptrs.add(i)))
        .collect()
}

/// Parse chunk hashes and sizes into a `Vec<(MerkleHash, u64)>`.
unsafe fn parse_chunk_pairs(
    chunk_hashes: *const *const c_char,
    chunk_sizes: *const u64,
    count: libc::size_t,
) -> Result<Vec<(MerkleHash, u64)>, String> {
    let mut pairs = Vec::with_capacity(count);
    for i in 0..count {
        let hash_str = c_str_opt(*chunk_hashes.add(i))
            .ok_or_else(|| format!("null chunk hash at index {i}"))?;
        let hash = MerkleHash::from_hex(&hash_str)
            .map_err(|e| format!("invalid chunk hash at index {i}: {e}"))?;
        let size = *chunk_sizes.add(i);
        pairs.push((hash, size));
    }
    Ok(pairs)
}

// ---------------------------------------------------------------------------
// Public C API — chunking
// ---------------------------------------------------------------------------

/// Split a raw byte buffer into content-defined chunks.
///
/// # Safety
/// `data` must be valid for `data_len` bytes.
/// The returned pointer must be freed with `xet_free_chunk_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_chunk_data(
    data: *const u8,
    data_len: libc::size_t,
) -> *mut XetChunkResult {
    if data.is_null() && data_len > 0 {
        return chunk_err("xet_chunk_data: null data pointer");
    }
    let slice = std::slice::from_raw_parts(data, data_len);
    let pairs = do_chunk_bytes(slice);
    chunk_pairs_to_c(pairs)
}

/// Split a file on disk into content-defined chunks.
///
/// # Safety
/// `file_path` must be a valid NUL-terminated UTF-8 string.
/// The returned pointer must be freed with `xet_free_chunk_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_chunk_file(file_path: *const c_char) -> *mut XetChunkResult {
    let path = match c_str_opt(file_path) {
        Some(p) => p,
        None => return chunk_err("xet_chunk_file: null file_path"),
    };

    let data = match std::fs::read(&path) {
        Ok(d) => d,
        Err(e) => return chunk_err(&format!("xet_chunk_file: failed to read {path}: {e}")),
    };

    let pairs = do_chunk_bytes(&data);
    chunk_pairs_to_c(pairs)
}

/// Release a `XetChunkResult` returned by `xet_chunk_data` or `xet_chunk_file`.
///
/// Passing `NULL` is a no-op.
///
/// # Safety
/// `result` must have been returned by this library and not previously freed.
#[no_mangle]
pub unsafe extern "C" fn xet_free_chunk_result(result: *mut XetChunkResult) {
    if result.is_null() {
        return;
    }
    let r = Box::from_raw(result);
    if !r.items.is_null() {
        let items = Vec::from_raw_parts(r.items, r.count, r.count);
        for item in items {
            if !item.hash.is_null() {
                drop(CString::from_raw(item.hash));
            }
        }
    }
    if !r.error.is_null() {
        drop(CString::from_raw(r.error));
    }
}

// ---------------------------------------------------------------------------
// Public C API — hash functions
// ---------------------------------------------------------------------------

/// Compute the Xet chunk hash of raw bytes.
///
/// # Safety
/// `data` must be valid for `data_len` bytes.
/// The returned pointer must be freed with `xet_free_hash_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_compute_chunk_hash(
    data: *const u8,
    data_len: libc::size_t,
) -> *mut XetHashResult {
    if data.is_null() && data_len > 0 {
        return hash_err("xet_compute_chunk_hash: null data pointer");
    }
    let slice = std::slice::from_raw_parts(data, data_len);
    let hash = compute_data_hash(slice);
    Box::into_raw(Box::new(XetHashResult {
        hash: opt_str_to_c(Some(&hash.hex())),
        error: ptr::null_mut(),
    }))
}

/// Compute the xorb hash from an ordered list of chunk hashes and sizes.
///
/// # Safety
/// All pointer arguments must be valid for the duration of the call.
/// The returned pointer must be freed with `xet_free_hash_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_compute_xorb_hash(
    chunk_hashes: *const *const c_char,
    chunk_sizes: *const u64,
    count: libc::size_t,
) -> *mut XetHashResult {
    match parse_chunk_pairs(chunk_hashes, chunk_sizes, count) {
        Ok(pairs) => {
            let h = xorb_hash(&pairs);
            Box::into_raw(Box::new(XetHashResult {
                hash: opt_str_to_c(Some(&h.hex())),
                error: ptr::null_mut(),
            }))
        }
        Err(e) => hash_err(&e),
    }
}

/// Compute the file hash from an ordered list of chunk hashes and sizes.
///
/// # Safety
/// All pointer arguments must be valid for the duration of the call.
/// The returned pointer must be freed with `xet_free_hash_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_compute_file_hash(
    chunk_hashes: *const *const c_char,
    chunk_sizes: *const u64,
    count: libc::size_t,
) -> *mut XetHashResult {
    match parse_chunk_pairs(chunk_hashes, chunk_sizes, count) {
        Ok(pairs) => {
            let h = file_hash(&pairs);
            Box::into_raw(Box::new(XetHashResult {
                hash: opt_str_to_c(Some(&h.hex())),
                error: ptr::null_mut(),
            }))
        }
        Err(e) => hash_err(&e),
    }
}

/// Compute the range hash from an ordered list of chunk hashes.
///
/// # Safety
/// All pointer arguments must be valid for the duration of the call.
/// The returned pointer must be freed with `xet_free_hash_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_compute_range_hash(
    chunk_hashes: *const *const c_char,
    count: libc::size_t,
) -> *mut XetHashResult {
    let hash_strings = c_strings_to_vec(chunk_hashes, count);
    let mut hashes = Vec::with_capacity(hash_strings.len());
    for (i, s) in hash_strings.iter().enumerate() {
        match MerkleHash::from_hex(s) {
            Ok(h) => hashes.push(h),
            Err(e) => return hash_err(&format!("invalid chunk hash at index {i}: {e}")),
        }
    }
    let h = range_hash_from_chunks(&hashes);
    Box::into_raw(Box::new(XetHashResult {
        hash: opt_str_to_c(Some(&h.hex())),
        error: ptr::null_mut(),
    }))
}

/// Release a `XetHashResult` returned by any `xet_compute_*_hash` function.
///
/// Passing `NULL` is a no-op.
///
/// # Safety
/// `result` must have been returned by this library and not previously freed.
#[no_mangle]
pub unsafe extern "C" fn xet_free_hash_result(result: *mut XetHashResult) {
    if result.is_null() {
        return;
    }
    let r = Box::from_raw(result);
    if !r.hash.is_null() {
        drop(CString::from_raw(r.hash));
    }
    if !r.error.is_null() {
        drop(CString::from_raw(r.error));
    }
}

// ---------------------------------------------------------------------------
// Public C API — xorb check
// ---------------------------------------------------------------------------

/// Deserialize an xorb from a byte slice and compute its hash and chunk info.
fn do_check_xorb(data: &[u8]) -> Result<*mut XetXorbCheckResult, String> {
    let mut cursor = std::io::Cursor::new(data);
    let (raw_data, boundaries) = deserialize_chunks(&mut cursor)
        .map_err(|e| format!("failed to deserialize xorb: {e}"))?;

    let num_chunks = boundaries.len().saturating_sub(1);
    let total_bytes = raw_data.len() as u64;

    let mut chunk_pairs: Vec<(MerkleHash, u64)> = Vec::with_capacity(num_chunks);
    for (start, end) in boundaries
        .iter()
        .take(num_chunks)
        .zip(boundaries.iter().skip(1))
    {
        let chunk = &raw_data[(*start as usize)..(*end as usize)];
        let hash = compute_data_hash(chunk);
        let size = (end - start) as u64;
        chunk_pairs.push((hash, size));
    }

    let computed_hash = xorb_hash(&chunk_pairs);

    let count = chunk_pairs.len();
    let mut items: Vec<XetChunkInfo> = chunk_pairs
        .iter()
        .map(|(hash, size)| XetChunkInfo {
            hash: opt_str_to_c(Some(&hash.hex())),
            size: *size,
        })
        .collect();
    let items_ptr = items.as_mut_ptr();
    std::mem::forget(items);

    let result = Box::into_raw(Box::new(XetXorbCheckResult {
        xorb_hash: opt_str_to_c(Some(&computed_hash.hex())),
        chunks: items_ptr,
        chunk_count: count,
        total_bytes,
        error: ptr::null_mut(),
    }));

    Ok(result)
}

/// Deserialize an xorb object from a raw byte buffer and compute its hash.
///
/// # Safety
/// `data` must be valid for `data_len` bytes.
/// The returned pointer must be freed with `xet_free_xorb_check_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_check_xorb_data(
    data: *const u8,
    data_len: libc::size_t,
) -> *mut XetXorbCheckResult {
    if data.is_null() && data_len > 0 {
        return xorb_check_err("xet_check_xorb_data: null data pointer");
    }
    let slice = std::slice::from_raw_parts(data, data_len);
    match do_check_xorb(slice) {
        Ok(ptr) => ptr,
        Err(e) => xorb_check_err(&e),
    }
}

/// Deserialize an xorb object from a file and compute its hash.
///
/// # Safety
/// `file_path` must be a valid NUL-terminated UTF-8 string.
/// The returned pointer must be freed with `xet_free_xorb_check_result`.
#[no_mangle]
pub unsafe extern "C" fn xet_check_xorb_file(file_path: *const c_char) -> *mut XetXorbCheckResult {
    let path = match c_str_opt(file_path) {
        Some(p) => p,
        None => return xorb_check_err("xet_check_xorb_file: null file_path"),
    };

    let data = match std::fs::read(&path) {
        Ok(d) => d,
        Err(e) => return xorb_check_err(&format!("xet_check_xorb_file: failed to read {path}: {e}")),
    };

    match do_check_xorb(&data) {
        Ok(ptr) => ptr,
        Err(e) => xorb_check_err(&e),
    }
}

/// Release a `XetXorbCheckResult` returned by `xet_check_xorb_data` or `xet_check_xorb_file`.
///
/// Passing `NULL` is a no-op.
///
/// # Safety
/// `result` must have been returned by this library and not previously freed.
#[no_mangle]
pub unsafe extern "C" fn xet_free_xorb_check_result(result: *mut XetXorbCheckResult) {
    if result.is_null() {
        return;
    }
    let r = Box::from_raw(result);
    if !r.xorb_hash.is_null() {
        drop(CString::from_raw(r.xorb_hash));
    }
    if !r.chunks.is_null() {
        let items = Vec::from_raw_parts(r.chunks, r.chunk_count, r.chunk_count);
        for item in items {
            if !item.hash.is_null() {
                drop(CString::from_raw(item.hash));
            }
        }
    }
    if !r.error.is_null() {
        drop(CString::from_raw(r.error));
    }
}
