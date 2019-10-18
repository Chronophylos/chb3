DEPS := analytics.go command.go main.go state.go

build: chb3

install: chb3
	install -Dm755 chb3 "/usr/bin"
	install -Dm644 chb3.service "/usr/lib/systemd/system"

chb3: $(DEPS)
	go build -v -i .

clean:
	-rm -f chb3

.PHONY: clean
