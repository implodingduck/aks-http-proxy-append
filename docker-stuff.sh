#!/bin/bash
docker stop akshttpproxyappend
docker rm akshttpproxyappend
docker build -t akshttpproxyappend .
docker run --name akshttpproxyappend -d -p 8443:8443 akshttpproxyappend