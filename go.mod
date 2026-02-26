module github.com/lnbotdev/cli

go 1.21

require (
	github.com/lnbotdev/go-sdk v0.1.0
	github.com/mdp/qrterminal/v3 v3.2.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	rsc.io/qr v0.2.0 // indirect
)

// Remove this replace directive once the go-sdk is published to a Go module proxy.
replace github.com/lnbotdev/go-sdk => ../go-sdk
