package main

import (
	"github.com/Berops/platform/proto/pb"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var cluster = &pb.Cluster{
	Name:       "TestName",
	Kubernetes: "v1.19.0",
	Network:    "192.168.2.0/24",
	PublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAAL2EmPjijvam+XCRMOThTzdDgqVc4+1Pu8mQH21CRAQGOsEyCfc8Qu6YN3wriEpgsARnmwWg3bqfWaP4qfAG6UfRra6QySmSYusVPDBfghxFQgSiZsBMFDy4EhsW89o+wHtN87Cvtys1Z2k+pcCTyIR4d6bK77eBjCFHvgCXNemHUtpHvcqI157rv/T4nB99aTWvRwGWwXX6l46iH7OD4m8UW/bZWBLSuWu9vSDFCrOUYDfl1s5KgjraXYIx2WW7CjqAxz5Zsk2zhiOiWk8igJWZJSP8iohq/TXrm2Zg9a8G4Bo73yH/XGQK3Y9a8HrDcaf7qx5lF1uRgkany7974k=",
	PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABlwAAAAdzc2gtcn\nNhAAAAAwEAAQAAAYEA0ecFSebBc3y4JwTnmmUYrussfBBlsR17mcuUQdT8aZTJdpx8egmW\nnRh2H9OiHcAm1QmXO8EXLRYeZO/emBBG/OhrssPlxdQa5aaReCQHZuNsLvvqPu0ZYzxR1k\n2fMzmjSg+D0+Rk+H488uQ9ya4dohWgPfdthdhKnb5xrF89l3yXf2N94vx2LW5B/S8KjY1o\nSS1LrglDowMYpOqOyqJ2UyXuyOou3hfmUUtKJZ2n86zkyODBuqzbJtYdeWFlU2mPJ6v1bw\nzWxKSpNt/NJFJx4s5ISdbEKa75nuBKPi8BILakX6XnU+LvcgzPv8PCmlBRqD9Qu1oJ5e4D\nagvwhACRn7GwRo77Qm8+9ycn/4qrQprtQo9w5R5gRFXRJ3vmj4pxg5KpdDkpJ8qidtjg2F\nMW4X4wiIvuCG03DgFGeyXdxP5grcfWi61YJMFgyqOPoCLci3rrEiDfVnEtHnvzUBnNUIyN\nOqFC44HCOK44yEnVrMWWePMbrULeUzrQCxayKggnAAAFkBJQb3wSUG98AAAAB3NzaC1yc2\nEAAAGBANHnBUnmwXN8uCcE55plGK7rLHwQZbEde5nLlEHU/GmUyXacfHoJlp0Ydh/Toh3A\nJtUJlzvBFy0WHmTv3pgQRvzoa7LD5cXUGuWmkXgkB2bjbC776j7tGWM8UdZNnzM5o0oPg9\nPkZPh+PPLkPcmuHaIVoD33bYXYSp2+caxfPZd8l39jfeL8di1uQf0vCo2NaEktS64JQ6MD\nGKTqjsqidlMl7sjqLt4X5lFLSiWdp/Os5Mjgwbqs2ybWHXlhZVNpjyer9W8M1sSkqTbfzS\nRSceLOSEnWxCmu+Z7gSj4vASC2pF+l51Pi73IMz7/DwppQUag/ULtaCeXuA2oL8IQAkZ+x\nsEaO+0JvPvcnJ/+Kq0Ka7UKPcOUeYERV0Sd75o+KcYOSqXQ5KSfKonbY4NhTFuF+MIiL7g\nhtNw4BRnsl3cT+YK3H1outWCTBYMqjj6Ai3It66xIg31ZxLR5781AZzVCMjTqhQuOBwjiu\nOMhJ1azFlnjzG61C3lM60AsWsioIJwAAAAMBAAEAAAGAQlsUEu6+DTJKTRuB1A9NpE54O6\ng7XaiCYHY5Ii6gtQfyQGrr9vB9CqCnBxyyTVFndUWY56z9FKW/ag1igxPyPRWEpnjDdKy+\n7AaiSiapqF8Q3jGJNedidTqmbGcRgvIfqtQIyr2TJfNSdT6uQcmnWIwZoj1MBFoCDKgd62\n4YXIFoqz7alx1UhrwqZE2wulsPssJ9AEGxfiEGc2wrQ+fkHBkLybwuoMtRZjW09PtIEwJ/\nOPnEhK0MgtSBcNPXYm0l37w7uu+rtiHrU6VmO5i0lp1RKYSFOxJ/H3iYN7qS+WIHLOaLqA\nh/9ehOdU6AH3ZYP8wA+jmHnXdLg5QXEB7Mp7aY4ivdk3rAiJzozr0CXD9DpD+RtlF82HSk\nQEr1fBtlP88C16rRL3CYAOzIXYc49ppFQQs5N7reK+ecNZ+m38wU914Kk9ZwaFe5vNPuja\ndQ5numPrhl6O0Z5UNOPm3Ra62Yc6ODiN713oJ4M5+DIkD8yzQ1dMQ5CHpWVlalMEdhAAAA\nwQCIAqiKTe85sHlCdG3QsOWV7oQIFB29csHcWAN73OKypQmV4Tz+Sz/ASR/WtwHpf9grNZ\n9XHVrW0m/9kTkvlhLfoHudO5paCUOpNsB5vdme0HsZtk3M1gXesAQqXprKydXcmibnRNFS\nbwzAxkkvQxjOpb9jXL1I2haPnvoQtIQhrYk56NT+F7i83g0DICWcvseamgkiq2vfxrxgC7\nTA19Uhi/FXkCLdsrXg3VgaaHDreQx0RjGDvQiERUIJH7UR/CIAAADBAO1GFKhm8/ATPgFF\nLoNJMld8VXN3/G4eeqCXGICzHRIGdU6n++A+0/hLCHla2z5wjO1AXW10RFj7XMN6Y/tdFj\n1sU19iDVLCMrhYS3UfsMvJlMr3XwbT5p/k99CJmzzCLvtfjJqUc3QQrhXF3IBhavHHAcx3\nXjlZyLjDohIAZTOgSNcTq1047CexoZdShSfBEeaGB4tU9WSsneeP32401a7nTIySPuFQdd\nhpJhJwyGZfSbp+dkcirtmKa04QqQ5iPwAAAMEA4nfsy/GlxHW478GxLRJwaOvJ4MDn0J+E\n4BchRwaSCT0wMRKSoHGFMSW8ujoUfCSr3hwTxiskfCHEJZ87X9CSwX+nLiQ1fOfW5fKado\na9859BC6IthforOrN3StjRbTbO6R69xYgISgH5gC2ROjg4evT9toOX2n3+bH+MxSEQ4TPs\ncyXsI/0a+GOfBH1odUtlGhaphWFFMURp0c6mmwrBS0+lWDtDBX97BANBC7Qlcm+z0Zh/CF\ng2PO2Shdo5wZAZAAAAGnNhbXVlbC5zdG9saWNueUBiZXJvcHMuY29t\n-----END OPENSSH PRIVATE KEY-----",
	Ips: map[string]*pb.Ip{
		"server2": &pb.Ip{
			Public:    "2.2.2.2",
			Private:   "192.168.2.2",
			IsControl: true,
		},
		"server1": &pb.Ip{
			Public:    "1.1.1.1",
			Private:   "192.168.2.1",
			IsControl: true,
		},
		"server3": &pb.Ip{
			Public:    "3.3.3.3",
			Private:   "192.168.2.3",
			IsControl: false,
		},
		"server4": &pb.Ip{
			Public:    "4.4.4.4",
			Private:   "192.168.2.4",
			IsControl: false,
		},
	},
}

func Test_genKubeOneConfig(t *testing.T) {
	var d data
	d.formatTemplateData(cluster)
	genKubeOneConfig("services/kube-eleven/server/kubeone.tpl", "services/kube-eleven/server/kubeone.yaml", d)
	if _, err := os.Stat("services/kube-eleven/server/kubeone.yaml"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		t.Error("kubeone.yaml file doesn't exist")
	}
}

func Test_createKeyFile(t *testing.T) {
	createKeyFile(cluster.GetPrivateKey(), "private.pem")
	if _, err := os.Stat("services/kube-eleven/server/private.pem"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		t.Error("private.pem file doesn't exist")
	}
}

func Test_runKubeOne(t *testing.T) {
	runKubeOne()
	require.NoError(t, nil)
}