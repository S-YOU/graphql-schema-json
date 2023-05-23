package main

var commonInitialisms = map[string]string{
	"Acl":   "ACL",
	"Api":   "API",
	"Ascii": "ASCII",
	"Cpu":   "CPU",
	"Css":   "CSS",
	"Csv":   "CSV",
	"Dns":   "DNS",
	"Eof":   "EOF",
	"Guid":  "GUID",
	"Html":  "HTML",
	"Http":  "HTTP",
	"Https": "HTTPS",
	"Icmp":  "ICMP",
	"Id":    "ID",
	"Ip":    "IP",
	"Json":  "JSON",
	"Kvk":   "KVK",
	"Lhs":   "LHS",
	"Pdf":   "PDF",
	"Pgp":   "PGP",
	"Qps":   "QPS",
	"Qr":    "QR",
	"Ram":   "RAM",
	"Rhs":   "RHS",
	"Rpc":   "RPC",
	"Sla":   "SLA",
	"Smtp":  "SMTP",
	"Sql":   "SQL",
	"Ssh":   "SSH",
	"Svg":   "SVG",
	"Tcp":   "TCP",
	"Tls":   "TLS",
	"Ttl":   "TTL",
	"Udp":   "UDP",
	"Ui":    "UI",
	"Uid":   "UID",
	"Uri":   "URI",
	"Url":   "URL",
	"Utf8":  "UTF8",
	"Uuid":  "UUID",
	"Vm":    "VM",
	"Xml":   "XML",
	"Xmpp":  "XMPP",
	"Xsrf":  "XSRF",
	"Xss":   "XSS",
}

var commonInitialismsRev = func() map[string]string {
	ret := make(map[string]string, len(commonInitialisms))
	for k, v := range commonInitialisms {
		ret[v] = k
	}
	return ret
}()

func ToGo(s string) string {
	for i := 5; i >= 2; i-- {
		l := len(s)
		if l >= i {
			if y := commonInitialisms[s[l-i:]]; y != "" {
				s = s[:l-i] + y
				return s
			}
		}
	}
	return s
}

func deInitialism(s string) string {
	for i := 5; i >= 2; i-- {
		l := len(s)
		if l >= i {
			if y := commonInitialismsRev[s[l-i:]]; y != "" {
				s = s[:l-i] + y
				return s
			}
		}
	}
	return s
}
