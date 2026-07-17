set GOOS=darwin&& set GOARCH=arm64&& go build -o bin\SMSSender_mac-arm .
set GOOS=darwin&& set GOARCH=amd64&& go build -o bin\SMSSender_mac-intel .
set GOOS=linux&& set GOARCH=arm64&& go build -o bin\SMSSender_linux-arm .
set GOOS=linux&& set GOARCH=amd64&& go build -o bin\SMSSender_linux-amd64 .
set GOOS=windows&& set GOARCH=amd64&& go build -o bin\SMSSender-amd64.exe .
set GOOS=windows&& set GOARCH=arm64&& go build -o bin\SMSSender-arm64.exe .
set GOOS=windows&& set GOARCH=amd64