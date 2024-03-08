## Knative Eventing

Knative Eventing is a set of building blocks for building event-driven architectures on Kubernetes. It provides abstractions and components to enable developers to build loosely coupled, event-driven microservices in a cloud-native environment. To facilitate handling event traffics, sonataflow operator provides a solution to automatically creating related Knative Eventing relevant resources.

### Prerequisite

1. Sonataflow Operator installed
2. Red Hat OpenShift Serverless operator is installed and Knative Eventing is initiated with a KnativeEventing CR.
3. a broker named `default` is created. Currently all Triggers created by Sonataflow operator will read events from `default`

### Description
1. in the following Sonataflow CR:
```yaml
apiVersion: sonataflow.org/v1alpha08
kind: SonataFlow
metadata:
...
spec:
  sink:
    ref:
      name: default
      namespace: greeting
      apiVersion: eventing.knative.dev/v1
      kind: Broker
  flow:
    events:
      - name: requestQuote
        type: kogito.sw.request.quote
        kind: produced
      - name: aggregatedQuotesResponse,
        type: kogito.loanbroker.aggregated.quotes.response,
        kind: consumed,
        source: /kogito/serverless/loanbroker/aggregator
...
```
* `spec.sink.ref` defines the sink that all created sinkBinding will use as the destination sink for producing events
* `spec.flow.events` lists all the events referenced in the workflow. Events with `produced` kind will trigger the creation of `SinkBindings` by the Sonataflow operator, while those labeled as `consumed` will lead to the generation of `Triggers`.

### Notes
* Knative resources is not watched by the operator, indicating that they won't undergo automatic reconciliation. This grants users the freedom to make updates at their discretion.

### Reference
Milestone in github:
https://github.com/apache/incubator-kie-kogito-serverless-operator/milestone/1