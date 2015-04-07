FROM google/golang

WORKDIR /tiny_mqs
ADD . /tiny_mqs/

CMD ["/tiny_mqs/tiny_mqs"]