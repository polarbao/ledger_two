#!/usr/bin/expect -f
set timeout 900

set password "QAZqaz2023@"
set user "polar"
set ip "192.168.0.115"

# 1. 传输源码部署包 (通过 SSH 管道传输，避开 SFTP/SCP 子系统限制)
puts "\n>>> (Expect) 传输源码包至群晖 /tmp 目录..."
spawn sh -c "cat ./ledger-two-deploy.tar.gz | ssh $user@$ip \"cat > /tmp/ledger-two-deploy.tar.gz\""
expect {
    "*yes/no*" {
        send "yes\r"
        exp_continue
    }
    -nocase "*password*" {
        send "$password\r"
    }
}
expect eof

# 2. 传输 NAS 启动脚本
puts "\n>>> (Expect) 传输部署脚本至群晖 /tmp 目录..."
spawn sh -c "cat nas_setup.sh | ssh $user@$ip \"cat > /tmp/nas_setup.sh\""
expect {
    -nocase "*password*" {
        send "$password\r"
    }
}
expect eof

# 3. 远程 SSH 并以 root 运行部署
puts "\n>>> (Expect) 连接群晖并运行 sudo 部署脚本 (需要执行编译构建，可能需要约 1-3 分钟)..."
spawn ssh -t $user@$ip "sudo bash /tmp/nas_setup.sh"
expect {
    -nocase "*password*" {
        send "$password\r"
        exp_continue
    }
    eof
}

puts "\n>>> (Expect) 部署流程全部结束！"
