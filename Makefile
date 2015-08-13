
build: hterm
	docker build -t progrium/envy .
	docker tag -f progrium/envy progrium/envy:local

dev:
	docker build -t progrium/envy:dev -f Dockerfile .
	docker run --rm --name envy.dev \
		-v /tmp/envy:/envy \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-p 8000:80 \
		-p 2222:22 \
		-e HOST_ROOT=/tmp/envy \
		progrium/envy:dev


test:
	test -f tests/envy.tgz || docker save progrium/envy > tests/envy.tgz
	test -f tests/dind.tgz || docker save progrium/dind > tests/dind.tgz
	test -f tests/alpine.tgz || docker save alpine > tests/alpine.tgz
	test -f tests/ubuntu.tgz || docker save ubuntu > tests/ubuntu.tgz
	docker rm -f envy.test &> /dev/null || true
	docker build -t envy-tests -f tests/Dockerfile .
	docker run -d --name envy.test --privileged envy-tests
	docker exec envy.test make test2

test2:
	#docker build -t progrium/envy .
	docker load -i /envy/tests/envy.tgz
	docker load -i /envy/tests/dind.tgz
	docker load -i /envy/tests/ubuntu.tgz
	docker load -i /envy/tests/alpine.tgz
	docker run -d \
		--net=host \
		-v /tmp/envy:/envy \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e ALLOWALL=true \
		-e HOST_ROOT=/tmp/envy \
		progrium/envy
	basht /envy/tests/*.bash

clean:
	docker rmi progrium/envy:local &> /dev/null || true
	rm -f tests/envy.tgz
	rm -f tests/dind.tgz
	rm -f tests/alpine.tgz
	rm -f tests/ubuntu.tgz

hterm:
	cd pkg/hterm && go generate
