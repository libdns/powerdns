package powerdns

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/mittwald/go-powerdns/apis/zones"
)

func TestPDNSClient(t *testing.T) {
	var dockerCompose string
	var ok bool
	doRun, _ := strconv.ParseBool(os.Getenv("PDNS_RUN_INTEGRATION_TEST"))
	if !doRun {
		t.Skip("skipping because PDNS_RUN_INTEGRATION_TEST was not set")
	}
	if dockerCompose, ok = which("docker-compose"); !ok {
		t.Skip("docker-compose is not present, skipping")
	}
	err := runCmd(dockerCompose, "rm", "-sfv")
	if err != nil {
		t.Fatalf("docker-compose failed: %s", err)
	}
	err = runCmd(dockerCompose, "down", "-v")
	if err != nil {
		t.Fatalf("docker-compose failed: %s", err)
	}
	err = runCmd(dockerCompose, "up", "-d")
	if err != nil {
		t.Fatalf("docker-compose failed: %s", err)
	}
	defer func() {
		if skipCleanup, _ := strconv.ParseBool(os.Getenv("PDNS_SKIP_CLEANUP")); !skipCleanup {
			runCmd(dockerCompose, "down", "-v")
		}
	}()

	time.Sleep(time.Second * 30) // give everything time to finish coming up
	z := zones.Zone{
		Name: "example.org.",
		Type: zones.ZoneTypeZone,
		Kind: zones.ZoneKindNative,
		ResourceRecordSets: []zones.ResourceRecordSet{
			{
				Name: "1.example.org.",
				Type: "A",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "127.0.0.1",
					},
					{
						Content: "127.0.0.2",
					},
					{
						Content: "127.0.0.3",
					},
				},
			},
			{
				Name: "1.example.org.",
				Type: "TXT",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "\"This is text\"",
					},
				},
			},
			{
				Name: "2.example.org.",
				Type: "A",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "127.0.0.4",
					},
					{
						Content: "127.0.0.5",
					},
					{
						Content: "127.0.0.6",
					},
				},
			},
			{
				Name: "example.org.",
				Type: "MX",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "10 mx1.example.org.",
					},
				},
			},
			{
				Name: "_https._tcp.example.org.",
				Type: "SRV",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "100 1 443 https.example.org.",
					},
				},
			},
		},
		Serial: 1,
		Nameservers: []string{
			"ns1.example.org.",
			"ns2.example.org.",
		},
	}
	p := &Provider{
		ServerURL: "http://localhost:8081",
		ServerID:  "localhost",
		APIToken:  "secret",
		Debug:     os.Getenv("PDNS_DEBUG"),
	}
	c, err := p.client()
	if err != nil {
		t.Fatalf("could not create client: %s", err)
	}
	_, err = c.Client.Zones().CreateZone(context.Background(), c.sID, z)
	if err != nil {
		t.Fatalf("failed to create test zone: %s", err)
	}

	for _, table := range []struct {
		name      string
		operation string
		zone      string
		Type      string
		records   []libdns.Record
		want      []string
	}{
		{
			name:      "Test Get Zone",
			operation: "records",
			zone:      "example.org.",
			records:   nil,
			Type:      "A",
			want:      []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3", "2:127.0.0.4", "2:127.0.0.5", "2:127.0.0.6"},
		},
		{
			name:      "Test Append Zone A record",
			operation: "append",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.7",
				},
			},
			want: []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3",
				"2:127.0.0.4", "2:127.0.0.5", "2:127.0.0.6", "2:127.0.0.7"},
		},
		{
			name:      "Test Append Zone TXT record",
			operation: "append",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: "\"This is also some text\"",
				},
			},
			want: []string{
				`1:"This is text"`,
				`1:"This is also some text"`,
			},
		},
		{
			name:      "Test Append Zone TXT record with weird formatting",
			operation: "append",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: "This is some weird text that isn't quoted",
				},
			},
			want: []string{
				`1:"This is text"`,
				`1:"This is also some text"`,
				`1:"This is some weird text that isn't quoted"`,
			},
		},
		{
			name:      "Test Append Zone TXT record with embedded quotes",
			operation: "append",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: `This is some weird text that "has embedded quoting"`,
				},
			},
			want: []string{`1:"This is text"`, `1:"This is also some text"`,
				`1:"This is some weird text that isn't quoted"`,
				`1:"This is some weird text that \"has embedded quoting\""`},
		},
		{
			name:      "Test Append Zone TXT record with unicode",
			operation: "append",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: `ç is equal to \195\167`,
				},
			},
			want: []string{`1:"This is text"`, `1:"This is also some text"`,
				`1:"This is some weird text that isn't quoted"`,
				`1:"This is some weird text that \"has embedded quoting\""`,
				`1:"ç is equal to \195\167"`,
			},
		},
		{
			name:      "Test Delete Zone TXT record with embedded quotes",
			operation: "delete",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: `This is some weird text that "has embedded quoting"`,
				},
			},
			want: []string{`1:"This is text"`, `1:"This is also some text"`,
				`1:"This is some weird text that isn't quoted"`,
				`1:"ç is equal to \195\167"`,
			},
		},
		{
			name:      "Test Delete Zone TXT record with unicode",
			operation: "delete",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: `ç is equal to \195\167`,
				},
			},
			want: []string{`1:"This is text"`, `1:"This is also some text"`,
				`1:"This is some weird text that isn't quoted"`,
			},
		},
		{
			name:      "Test Delete Zone",
			operation: "delete",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.5",
				},
			},
			want: []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3", "2:127.0.0.4", "2:127.0.0.6", "2:127.0.0.7"},
		},
		{
			name:      "Test Append and Add Zone",
			operation: "append",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.8",
				},
				{
					Name:  "3",
					Type:  "A",
					Value: "127.0.0.9",
				},
			},
			want: []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3",
				"2:127.0.0.4", "2:127.0.0.6", "2:127.0.0.7", "2:127.0.0.8",
				"3:127.0.0.9"},
		},
		{
			name:      "Test Set",
			operation: "set",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.1",
				},
				{
					Name:  "1",
					Type:  "A",
					Value: "127.0.0.1",
				},
			},
			want: []string{"1:127.0.0.1", "2:127.0.0.1", "3:127.0.0.9"},
		},
		{
			name:      "Test Get Zone MX records",
			operation: "records",
			zone:      "example.org.",
			records:   nil,
			Type:      "MX",
			want:      []string{":mx1.example.org.:10"},
		},
		{
			name:      "Test Append Zone MX record",
			operation: "append",
			zone:      "example.org.",
			Type:      "MX",
			records: []libdns.Record{
				{
					Name:     "",
					Type:     "MX",
					Value:    "mx2.example.org.",
					Priority: 20,
				},
				{
					Name:     "",
					Type:     "MX",
					Value:    "mx3.example.org.",
					Priority: 30,
				},
			},
			want: []string{
				":mx1.example.org.:10",
				":mx2.example.org.:20",
				":mx3.example.org.:30",
			},
		},
		{
			name:      "Test Delete Zone MX record",
			operation: "delete",
			zone:      "example.org.",
			Type:      "MX",
			records: []libdns.Record{
				{
					Name:     "",
					Type:     "MX",
					Value:    "mx2.example.org.",
					Priority: 20,
				},
			},
			want: []string{
				":mx1.example.org.:10",
				":mx3.example.org.:30",
			},
		},
		{
			name:      "Test Set Zone MX record",
			operation: "set",
			zone:      "example.org.",
			Type:      "MX",
			records: []libdns.Record{
				{
					Name:     "",
					Type:     "MX",
					Value:    "mx2.example.org.",
					Priority: 20,
				},
			},
			want: []string{
				":mx2.example.org.:20",
			},
		},
		{
			name:      "Test Get Zone SRV records",
			operation: "records",
			zone:      "example.org.",
			records:   nil,
			Type:      "SRV",
			want: []string{
				"_https._tcp:1 443 https.example.org.:100",
			},
		},
		{
			name:      "Test Append Zone SRV record",
			operation: "append",
			zone:      "example.org.",
			Type:      "SRV",
			records: []libdns.Record{
				{
					Name:     "_imaps._tcp",
					Type:     "SRV",
					Value:    "1 993 imaps.example.org.",
					Priority: 200,
				},
				{
					Name:     "_pop3s._tcp",
					Type:     "SRV",
					Value:    "1 995 pop3s.example.org.",
					Priority: 300,
				},
			},
			want: []string{
				"_https._tcp:1 443 https.example.org.:100",
				"_imaps._tcp:1 993 imaps.example.org.:200",
				"_pop3s._tcp:1 995 pop3s.example.org.:300",
			},
		},
		{
			name:      "Test Delete Zone SRV record",
			operation: "delete",
			zone:      "example.org.",
			Type:      "SRV",
			records: []libdns.Record{
				{
					Name:     "_imaps._tcp",
					Type:     "SRV",
					Value:    "1 993 imaps.example.org.",
					Priority: 200,
				},
			},
			want: []string{
				"_https._tcp:1 443 https.example.org.:100",
				"_pop3s._tcp:1 995 pop3s.example.org.:300",
			},
		},
		{
			name:      "Test Set Zone SRV record",
			operation: "set",
			zone:      "example.org.",
			Type:      "SRV",
			records: []libdns.Record{
				{
					Name:     "_https._tcp",
					Type:     "SRV",
					Value:    "1 8443 https-alt.example.org.",
					Priority: 50,
				},
			},
			want: []string{
				"_https._tcp:1 8443 https-alt.example.org.:50",
				"_pop3s._tcp:1 995 pop3s.example.org.:300",
			},
		},
	} {
		t.Run(table.name, func(t *testing.T) {
			var err error
			switch table.operation {
			case "records":
				// fetch below
			case "append":
				_, err = p.AppendRecords(context.Background(), table.zone, table.records)
			case "set":
				_, err = p.SetRecords(context.Background(), table.zone, table.records)
			case "delete":
				_, err = p.DeleteRecords(context.Background(), table.zone, table.records)
			}

			if err != nil {
				t.Errorf("failed to %s records: %s", table.operation, err)
				return
			}

			// Fetch the zone
			recs, err := p.GetRecords(context.Background(), table.zone)
			if err != nil {
				t.Errorf("error fetching zone")
				return
			}
			var have []string
			for _, rr := range recs {
				if rr.Type != table.Type {
					continue
				}

				switch rr.Type {
				case "MX", "SRV":
					have = append(have, fmt.Sprintf("%s:%s:%d", rr.Name, rr.Value, rr.Priority))
				default:
					have = append(have, fmt.Sprintf("%s:%s", rr.Name, rr.Value))
				}
			}

			sort.Strings(have)
			sort.Strings(table.want)
			if !reflect.DeepEqual(have, table.want) {
				t.Errorf("assertion failed: have: %#v want %#v", have, table.want)
			}

		})
	}

}

func which(cmd string) (string, bool) {
	pth, err := exec.LookPath(cmd)
	if err != nil {
		return "", false
	}
	return pth, true
}

func runCmd(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
