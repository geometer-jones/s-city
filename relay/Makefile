.PHONY: test cover vuln ci

test:
	go test ./...

cover:
	go test ./... -coverpkg=./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

vuln:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

ci: test cover vuln
