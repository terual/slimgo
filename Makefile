include $(GOROOT)/src/Make.inc

all: slimgo

slimgo: alsa
	$(GC) -o _go_.$O -I_obj slimproto.go slimaudio.go slimbuffer.go bufio.go main.go
	$(LD) -o $@ -L_obj _go_.$O 
	@echo "Done. Executable is: $@"

alsa:
	gomake -C alsa-go

clean:
	gomake -C alsa-go clean
	rm -rf *.[$(OS)o] *.a [$(OS)].out _obj _test _testmain.go slimgo

format:
	find . -type f -name '*.go' -exec gofmt -w {} \;


