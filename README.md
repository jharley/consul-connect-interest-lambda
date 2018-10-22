# consul-connect-interest-lambda

This repository contains an example of an [AWS Lambda](https://aws.amazon.com/lambda/) function written in Golang and making use of the [Consul Connect](https://www.consul.io/docs/connect/index.html) Native Integration.

## Background

The Lambda function itself is a rather contrived example, of an account interest calculation function which runs on a set interval within the company. Interest balances are stored in a Redis database which is registered as a Consul Connect service and access to which is controlled by Consul's [Service Access Graph](https://www.consul.io/docs/connect/intentions.html).
