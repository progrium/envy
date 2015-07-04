
build: hterm
	docker build -t progrium/envy .

dev: build
	docker run --rm --name envy.dev \
		-v /tmp/data:/data \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-p 8000:80 \
		-p 2222:22 \
		-e HOST_DATA=/tmp/data \
		progrium/envy

hterm:
	cd pkg/hterm && go generate
