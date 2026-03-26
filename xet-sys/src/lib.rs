//! xet-sys — Rust C API static library for the Go hf_xet package.
//!
//! This crate exposes the five functions declared in `hf_xet/hf_xet.h` as a
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

// ---------------------------------------------------------------------------
// C struct mirrors — must match hf_xet/hf_xet.h exactly (#[repr(C)])
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
