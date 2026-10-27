NAME=woaa
BUILD_DIR=build
GOBUILD=CGO_ENABLED=0 go build -trimpath -ldflags '-w -s -buildid='

.PHONY: all clean default fe linux-amd64 release-linux-amd64

default: fe
	$(GOBUILD) -o $(BUILD_DIR)/$(NAME) main.go

clean:
	rm -rf $(BUILD_DIR)

fe:
	cd wechat-official-account-admin-fe && pnpm install && pnpm build

linux-amd64: fe
	mkdir -p $(BUILD_DIR)
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(NAME)-linux-amd64 main.go

release-linux-amd64: linux-amd64
	mkdir -p $(BUILD_DIR)/release
	cp $(BUILD_DIR)/$(NAME)-linux-amd64 $(BUILD_DIR)/release/$(NAME)
	cp -r config $(BUILD_DIR)/release/
	cp -r atlas $(BUILD_DIR)/release/
	cp .env.example $(BUILD_DIR)/release/
	cp README.md $(BUILD_DIR)/release/
	tar -czf $(BUILD_DIR)/$(NAME)-linux-amd64.tar.gz -C $(BUILD_DIR)/release .
