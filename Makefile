DEPS := analytics.go command.go main.go state.go VERSION

build: chb3

chb3: $(DEPS)
	govvv build -v -i .

clean:
	-rm -f chb3

.PHONY: clean
