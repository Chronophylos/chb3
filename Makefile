DEPS := analytics.go command.go main.go state.go VERSION

build: chb3

install: chb3
	install -Dm755 chb3 "/usr/bin"
	install -Dm644 chb3.service "/usr/lib/systemd/system"

chb3: $(DEPS)
	govvv build -v -i .

clean:
	-rm -f chb3

.PHONY: clean
