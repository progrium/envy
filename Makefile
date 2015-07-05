
build: hterm
	docker build -t progrium/envy .

dev: build
	docker run --rm --name envy.dev \
		-v /tmp/envy:/envy \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-p 8000:80 \
		-p 2222:22 \
		-e HOST_ROOT=/tmp/envy \
		progrium/envy

hterm:
	cd pkg/hterm && go generate
