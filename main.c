#include <stdio.h>
#include <sys/socket.h>
#include <sys/un.h>

#define SV_SOCK_PATH "/tmp/wtmp.sock"

#define err_exit(out) printf(out); printf("%n"); exit(EXIT_FAILURE);
#define err_exitf(fmt, args) printf(fmt, args); printf("%n"); exit(EXIT_FAILURE);

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>


struct sock {
    char *path;
    int sfd;
};

struct sock create_socket() {
    struct sockaddr_un addr;

    // Create a new server socket with domain: AF_UNIX, type: SOCK_STREAM, protocol: 0
    int sfd = socket(AF_UNIX, SOCK_STREAM, 0);
    printf("Server socket fd = %d\n", sfd);

    if (sfd == -1) {
        err_exit("socket");
    }

    char *path = tmpnam(NULL);

    if (strlen(path) > sizeof(addr.sun_path) - 1) {
        err_exitf("Server socket path too long: %s", SV_SOCK_PATH);
    }

    memset(&addr, 0, sizeof(struct sockaddr_un));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, path, sizeof(addr.sun_path) - 1);

    if (bind(sfd, (struct sockaddr *) &addr, sizeof(struct sockaddr_un)) == -1) {
        err_exit("bind");
    }
    struct sock ret;
    ret.path = malloc(strlen(path));
    strcpy(ret.path, path);
    ret.sfd = sfd;
    return ret;

}

void send_message(char *msg, size_t len) {
    struct sockaddr_un addr;

    // Create a new client socket with domain: AF_UNIX, type: SOCK_STREAM, protocol: 0
    int sfd = socket(AF_UNIX, SOCK_STREAM, 0);
    printf("Client socket fd = %d\n", sfd);

    // Make sure socket's file descriptor is legit.
    if (sfd == -1) {
        err_exit("socket");
    }

    //
    // Construct server address, and make the connection.
    //
    memset(&addr, 0, sizeof(struct sockaddr_un));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, SV_SOCK_PATH, sizeof(addr.sun_path) - 1);

    // Connects the active socket referred to be sfd to the listening socket
    // whose address is specified by addr.
    printf("aaa\n");
    if (connect(sfd, (struct sockaddr *) &addr,
                sizeof(struct sockaddr_un)) == -1) {
        err_exitf("connect %d", errno);
    }
    //
    // Copy stdin to socket.
    //
    if (write(sfd, msg, len) != len) {
        err_exit("didn't write enough")
    }

    // Closes our socket; server sees EOF.
}

void listen_socket(int sfd) {}

int main(int argc, char *argv[]) {
    struct sock socket = create_socket();
    send_message(socket.path, strlen(socket.path));
    listen_socket(socket.sfd);
    free(socket.path);
}
