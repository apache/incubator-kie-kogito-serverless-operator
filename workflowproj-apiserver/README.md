# Workflowproj API Server

This is a **tech preview** module to enable clients push ZIP files containing [SonataFlow](https://sonataflow.org/) projects, 
usually made with a `sw.json` file and resources like an OpenAPI file. 
[Click here](https://github.com/krisv/kogito-serverless-workflow-helloworld/tree/main/src/main/resources) to see an example.

> **HEADS UP**! This module is not meant to be used in production yet.

Module based on [sample-apiserver](https://github.com/kubernetes/sample-apiserver).

## Deploying the API Server on Minikube

Prerequisities:

1. Minikube installed with registry add-on enabled
2. Go path configured
3. Docker
4. kubectl and kustomize installed

From this directory, run:

```shell
make deploy-example-minikube
```

## Running the SonataFlow Operator locally

You need the SonataFlow Operator running locally to deploy your zipped project. 
In a separate terminal, run:

```shell
make run-operator-locally
```

## Running the example

You should be able to run the example on your cluster.

Create a namespace to deploy your example:

```shell
kubectl create ns flow-example
```

Deploy an instance of a `SonataFlowProj` so we enable the API Server to receive the zip package.

```shell
kubectl apply -f artifacts/workflowproj/sonataflowproj.yaml -n flow-example
```

In order to enable the API Server to receive your requests without bothering to authentication, in a separate terminal run:

```shell
kubectl proxy --port=8080 &
```

You can now deploy the workflow project in a zip file:

```shell
curl -H "Content-type: application/zip" --trace-ascii debugcurl.log \
  --data-binary @pkg/generator/testdata/workflow-service.zip \
  http://127.0.0.1:8080/apis/workflowproj.sonataflow.org/v1alpha08/namespaces/flow-example/sonataflowprojinstances/my-first-sonataflow/instantiate
```

You may follow the deployment status running:

```shell
kubectl get workflow -n flow-example
```

The `SonataFlowProj` instance holds the information of the deployment:

```shell
kubectl describe sonataflowproj -n flow-example
Name:         my-first-sonataflow
Namespace:    flow-example
Labels:       <none>
Annotations:  <none>
API Version:  sonataflow.org/v1alpha08
Kind:         SonataFlowProj
Metadata:
  Creation Timestamp:  2023-11-23T19:04:42Z
  Generation:          1
  Resource Version:    418150
  UID:                 ff85f1e9-6864-4f44-8724-8548371fe327
Status:
  Objects Ref:
    Properties:
      Name:  my-first-sonataflow-props
    Resources:
      Name:  01-my-first-sonataflow-resources
    Workflow:
      Name:  my-first-sonataflow
Events:      <none>
```

## Clean up

Delete the namespace where we deployed the example:

```shell
kubectl delete ns flow-example
```

Undeploy the API Server:

```shell
make undeploy-example-minikube
```

Finally, you can stop the operator running in the other terminal.
