
setup() {
  rm -rf /envy/*
  docker pull progrium/dind:latest
  expect <<-EOF
    set timeout 1
    spawn ssh test@localhost
    expect {
      timeout   { exit 1 }
      eof       { exit 1 }
      "Are you sure" {
        send "yes\r"
        sleep 1
        exit 0
      }
    }
    exit 1
EOF
  echo
}
setup

T_session-reload() {
  expect <<-EOF
    set timeout 1
    spawn ssh reload@localhost
    expect {
      timeout           { exit 1 }
      eof               { exit 1 }
      "Building environment" {
        expect "root@reload"
        send "rm -rf /usr\r"
        expect "root@reload"
        send "reload\r"
        expect "root@reload"
        send "ls -1 usr\r"
        expect {
          "usr" { exit 0 }
        }
      }
    }
    exit 1
EOF
}

T_environ-rebuild() {
  expect <<-EOF
    set timeout 1
    spawn ssh rebuild@localhost
    expect {
      timeout           { exit 1 }
      eof               { exit 1 }
      "Building environment" {
        expect "root@rebuild"
        send "echo FROM alpine > /env/Dockerfile\r"
        expect "root@rebuild"
        send "rebuild\r"
        expect {
          "/ #" { exit 0 }
        }
      }
    }
    exit 1
EOF
}
