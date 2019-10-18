DEPS := analytics.go command.go main.go state.go

build: chb3

install: chb3
	install -Dm755 chb3 "/usr/bin/"
	install -Dm644 chb3.service "/usr/lib/systemd/system/"

strip: chb3
	@echo "Strip symbols"
	strip -v -s chb3

chb3: $(DEPS)
	go build -v -i .

clean:
	-rm -f chb3

.PHONY: clean
