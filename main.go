package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"inet.af/netaddr"
	"tailscale.com/client/tailscale"
)

type DNSDomain struct {
	Domain string
	Sub    string
}

func (d DNSDomain) BuildHostname(host string) string {
	return strings.ToLower(host) + "." + d.String()
}

func (d DNSDomain) String() string {
	suffix := d.Domain
	if len(d.Sub) > 0 {
		suffix = d.Sub + "." + d.Domain
	}
	return strings.ToLower(suffix)
}

type tailHost struct {
	Name string
	IP   netaddr.IP
}

func (t tailHost) RecordType() string {
	if t.IP.Is6() {
		return "AAAA"
	}
	return "A"
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "flags"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func sanitizeHost(s string) string {
	return strings.Replace(s, " ", "-", -1)
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	dd := DNSDomain{}
	var removeAll, removeUnused bool
	var alias arrayFlags
	flag.StringVar(&dd.Domain, "zone", "", "zone, ex. example.com")
	flag.StringVar(&dd.Sub, "subdomain", "", "subdomain to use, e.g. 'wg' will make dns records as <tailscale host>.wg.example.com")
	flag.BoolVar(&removeUnused, "remove-orphans", false, "remove DNS records that are not in tailscale")
	flag.BoolVar(&removeAll, "remove-all", false, "remove all tailscale dns records")
	flag.Var(&alias, "alias", "alias records")
	flag.Parse()

	aliasMap := make(map[string][]string, 0)
	for _, a := range alias {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 {
			host := parts[0]
			aliases := strings.Split(parts[1], ",")
			if len(aliases) > 0 {
				aliasMap[host] = aliases
			}
		}
	}

	ctx := context.Background()
	status, err := tailscale.Status(ctx)
	if err != nil {
		log.Fatal(err)
	}
	hostList := make([]tailHost, 0, 1+len(status.Peer))
	for _, ip := range status.Self.TailscaleIPs {
		hostList = append(hostList, tailHost{
			Name: sanitizeHost(status.Self.HostName),
			IP:   ip,
		})
	}
	for _, peer := range status.Peer {
		for _, ip := range peer.TailscaleIPs {
			hostList = append(hostList, tailHost{
				Name: sanitizeHost(peer.HostName),
				IP:   ip,
			})
		}
	}

	aliasList := make([]tailHost, 0)
	for _, host := range hostList {
		if aliases, ok := aliasMap[host.Name]; ok {
			for _, a := range aliases {
				aliasList = append(aliasList, tailHost{
					Name: sanitizeHost(a),
					IP:   host.IP,
				})
			}
		}
	}
	hostList = append(hostList, aliasList...)

	api, err := cloudflare.NewWithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(dd.Domain)
	if err != nil {
		log.Fatal(err)
	}

	currentRecords, err := api.DNSRecords(ctx, zoneID, cloudflare.DNSRecord{})
	if err != nil {
		log.Fatal(err)
	}

	currentRecordMap := make(map[string]cloudflare.DNSRecord, len(currentRecords))
	for _, r := range currentRecords {
		currentRecordMap[strings.ToLower(r.Type+r.Name)] = r
	}

	if removeAll {
		for _, r := range currentRecords {
			if (r.Type == "A" || r.Type == "AAAA") && strings.HasSuffix(r.Name, dd.String()) {
				log.Printf("removing record with name %s, ip %s", r.Name, r.Content)
				if err := api.DeleteDNSRecord(ctx, zoneID, r.ID); err != nil {
					log.Fatal(err)
				}
			}
		}
		return
	}

	tHostMap := make(map[string]struct{}, len(hostList))
	for _, t := range hostList {
		recordType := t.RecordType()
		recordName := dd.BuildHostname(t.Name)
		cfDnsRecord := cloudflare.DNSRecord{
			Type:    recordType,
			Name:    recordName,
			Content: t.IP.String(),
			TTL:     1,
		}
		action := "updated"
		var err error
		if r, exists := currentRecordMap[strings.ToLower(recordType+recordName)]; exists {
			err = api.UpdateDNSRecord(ctx, zoneID, r.ID, cfDnsRecord)
		} else {
			action = "created"
			_, err = api.CreateDNSRecord(ctx, zoneID, cfDnsRecord)
		}
		if err != nil {
			log.Fatalf("unable to create record %v. err: %v", cfDnsRecord, err)
		}
		log.Printf("%s dns record type %s, host %s, ip %s", action, recordType, recordName, t.IP)
		tHostMap[strings.ToLower(recordType+recordName)] = struct{}{}
	}

	if removeUnused {
		for _, r := range currentRecordMap {
			if strings.HasSuffix(r.Name, dd.String()) {
				if _, exists := tHostMap[strings.ToLower(r.Type+r.Name)]; !exists {
					log.Printf("removing record with name %s, ip %s", r.Name, r.Content)
					if err := api.DeleteDNSRecord(ctx, zoneID, r.ID); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}
