build: chb3

chb3: **/*.go
	go build -v -i .

clean:
	-rm -f chb3

.PHONY: clean
