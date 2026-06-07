fn main() {
    cc::Build::new()
        .cpp(true)
        .flag_if_supported("-std=c++17")
        .file("packet_capture.cpp")
        .compile("packet_capture");
}
