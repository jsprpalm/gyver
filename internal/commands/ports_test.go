package commands

import "testing"

func TestParseSS(t *testing.T) {
	out := `Netid State  Recv-Q Send-Q Local Address:Port Peer Address:Port Process
tcp   LISTEN 0      128    0.0.0.0:22        0.0.0.0:*          users:(("sshd",pid=812,fd=3))
udp   UNCONN 0      0      127.0.0.53%lo:53  0.0.0.0:*          users:(("systemd-resolve",pid=701,fd=12))
`
	rows := parseSS(out)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Proto != "tcp" || rows[0].Address != "0.0.0.0:22" {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[0].Process != "sshd (pid 812)" {
		t.Errorf("row0 process = %q, want sshd (pid 812)", rows[0].Process)
	}
	if rows[1].Process != "systemd-resolve (pid 701)" {
		t.Errorf("row1 process = %q", rows[1].Process)
	}
}

func TestParseLsof(t *testing.T) {
	out := `COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
sshd      812 root    3u  IPv4  12345      0t0  TCP *:22 (LISTEN)
Dropbox  1234 jesper 30u  IPv4  54321      0t0  TCP 127.0.0.1:17600 (LISTEN)
ntpd      99  root   20u  IPv4  11111      0t0  UDP *:123
chrome   500 jesper 88u  IPv4  22222      0t0  TCP 10.0.0.2:54321->1.2.3.4:443 (ESTABLISHED)
`
	rows := parseLsof(out)
	// Two TCP LISTEN + one UDP = 3; the ESTABLISHED row is excluded.
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3: %+v", len(rows), rows)
	}
	if rows[0].Proto != "TCP" || rows[0].Address != "*:22" {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[0].Process != "sshd (pid 812)" {
		t.Errorf("row0 process = %q", rows[0].Process)
	}
}
