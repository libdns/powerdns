package main

import (
	"context"

	"github.com/libdns/libdns"

	"github.com/hostpoint-ag/libdns-powerdns"
)

func main() {
	p := &powerdns.Provider{
		ServerURL: "http://localhost", // required
		ServerID:  "localhost",        // if left empty, defaults to localhost.
		APIToken:  "asdfasdfasdf",     // required
	}

	_, err := p.AppendRecords(context.Background(), "example.org.", []libdns.Record{
		{
			Name:  "_acme_whatever",
			Type:  "TXT",
			Value: "123456",
		},
	})
	if err != nil {
		panic(err)
	}

}
