# cloudflare-tailscale-dns

Small go program that sets cloudflare dns records for tailscale peers. This can
be used to add a search domain to resolv.conf as an alternative to tailscale's
magic dns feature.

### setup:

1. `go install github.com/sclem/cloudflare-tailscale-dns`

2. Obtain a cloudflare API token with DNS edit permissions and set
   `CLOUDFLARE_API_TOKEN` in your environment.

3. Run this program on a tailscale peer node.

### Usage:

`cloudflare-tailscale-dns -zone example.com -subdomain wg`

Optionally add `-remove-orphans` flag to remove any orphaned dns records from
the domain.

`-alias` flag (can be specified multiple times) creates duplicate dns records
for hosts, ex:

`cloudflare-tailscale-dns -zone example.com -subdomain wg -alias myhost=h1,h2`

will create dns entries for `myhost.wg.example.com`, `h1.wg.example.com`
`h2.wg.example.com`


`-remove-all` flag to remove all A/AAAA dns records under `<zone>.<subdomain>`.

`getent hosts <tailscale peer>.wg.example.com` to test.
