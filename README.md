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

`-remove-all` flag to remove all A/AAAA dns records under `<zone>.<subdomain>`.

`getent hosts <tailscale peer>.wg.example.com` to test.
