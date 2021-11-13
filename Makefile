.PHONY: list
list:
	@LC_ALL=C $(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

build-prepare:
	rm -rf bin
	mkdir bin

clean:
	rm -rf bin
	rm env/cockroach_bkp

build:
	make build-prepare
	cd src; go build -o cockroach_bkp; strip cockroach_bkp
	mv src/cockroach_bkp bin/
	cp bin/cockroach_bkp env/ || echo "env/ does not exist"

zbuild:
	make build
	upx --lzma bin/cockroach_bkp
