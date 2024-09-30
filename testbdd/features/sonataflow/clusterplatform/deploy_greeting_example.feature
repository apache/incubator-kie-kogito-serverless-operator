Feature: Deploy SonataFlowClusterPlatform with default SonataFlowPlatform

  Background:
    Given Namespace is created
    When SonataFlow Operator is deployed
    When SonataFlowClusterPlatform is deployed
    When SonataFlowPlatform deployed