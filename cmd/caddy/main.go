// Command caddy builds the Caddy distribution used by this repository.
package main

import (
	_ "time/tzdata"

	_ "github.com/TnZzZHlp/caddy/dynamic_proxy"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	_ "github.com/caddyserver/replace-response"
)

func main() {
	caddycmd.Main()
}
