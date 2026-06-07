#include <arpa/inet.h>
#include <atomic>
#include <cstring>
#include <fstream>
#include <iostream>
#include <linux/if_packet.h>
#include <net/ethernet.h>
#include <net/if.h>
#include <netinet/ip.h>
#include <netinet/udp.h>
#include <netinet/tcp.h>
#include <string>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <thread>
#include <unistd.h>

static std::atomic<bool> g_running(false);
static std::atomic<uint64_t> g_packet_count(0);
static std::thread g_capture_thread;
static std::string g_log_file;

extern "C" int start_packet_capture(const char* iface, const char* log_path) {
    if (g_running.load()) {
        return 0;
    }
    g_running.store(true);
    g_packet_count.store(0);
    g_log_file = log_path ? log_path : "../system/runtime/packet_capture.log";
    std::string intf = iface ? iface : "wlan0";

    g_capture_thread = std::thread([intf]() {
        int sock = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
        if (sock < 0) {
            return;
        }

        struct ifreq ifr;
        std::memset(&ifr, 0, sizeof(ifr));
        std::strncpy(ifr.ifr_name, intf.c_str(), IFNAMSIZ - 1);
        if (ioctl(sock, SIOCGIFINDEX, &ifr) == -1) {
            close(sock);
            return;
        }

        struct sockaddr_ll saddr;
        std::memset(&saddr, 0, sizeof(saddr));
        saddr.sll_family = AF_PACKET;
        saddr.sll_ifindex = ifr.ifr_ifindex;
        saddr.sll_protocol = htons(ETH_P_ALL);
        if (bind(sock, reinterpret_cast<struct sockaddr*>(&saddr), sizeof(saddr)) == -1) {
            close(sock);
            return;
        }

        std::ofstream logfile(g_log_file, std::ios::app);
        if (!logfile.is_open()) {
            logfile.open("/tmp/ap1_packet_capture.log", std::ios::app);
        }

        constexpr size_t buffer_size = 65536;
        char buffer[buffer_size];
        while (g_running.load()) {
            ssize_t packet_len = recv(sock, buffer, buffer_size, 0);
            if (packet_len <= 0) {
                continue;
            }
            g_packet_count.fetch_add(1);
            if (static_cast<size_t>(packet_len) < sizeof(struct ethhdr) + sizeof(struct iphdr)) {
                continue;
            }
            auto* eth = reinterpret_cast<struct ethhdr*>(buffer);
            if (ntohs(eth->h_proto) != ETH_P_IP) {
                continue;
            }
            auto* ip = reinterpret_cast<struct iphdr*>(buffer + sizeof(struct ethhdr));
            char src[INET_ADDRSTRLEN] = {0};
            char dst[INET_ADDRSTRLEN] = {0};
            inet_ntop(AF_INET, &ip->saddr, src, sizeof(src));
            inet_ntop(AF_INET, &ip->daddr, dst, sizeof(dst));
            std::string proto = std::to_string(static_cast<int>(ip->protocol));
            logfile << src << " -> " << dst << " proto=" << proto << " len=" << packet_len << "\n";
            logfile.flush();
        }

        close(sock);
    });

    return 0;
}

extern "C" void stop_packet_capture() {
    g_running.store(false);
    if (g_capture_thread.joinable()) {
        g_capture_thread.join();
    }
}

extern "C" uint64_t packet_capture_count() {
    return g_packet_count.load();
}
