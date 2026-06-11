#!/usr/bin/expect -f
set timeout 60

set password "QAZqaz2023@"
set user "polar"
set ip "192.168.0.115"

spawn ssh -t $user@$ip "export PATH=/volume1/@appstore/ContainerManager/usr/bin:\$PATH; sudo docker ps -a; echo '=== LOGS ==='; sudo docker logs ledger-two"
expect {
    -nocase "*password*" {
        send "$password\r"
        exp_continue
    }
    eof
}
