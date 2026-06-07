use std::ffi::CString;
use std::os::raw::{c_char, c_int};

extern "C" {
    fn start_packet_capture(iface: *const c_char, log_path: *const c_char) -> c_int;
    fn stop_packet_capture();
    fn packet_capture_count() -> u64;
}

pub fn start(iface: &str, log_path: &str) -> Result<(), String> {
    let iface_c = CString::new(iface).map_err(|e| e.to_string())?;
    let log_c = CString::new(log_path).map_err(|e| e.to_string())?;
    let result = unsafe { start_packet_capture(iface_c.as_ptr(), log_c.as_ptr()) };
    if result == 0 {
        Ok(())
    } else {
        Err("failed to start packet capture".into())
    }
}

pub fn stop() {
    unsafe { stop_packet_capture() }
}

pub fn count() -> u64 {
    unsafe { packet_capture_count() }
}
