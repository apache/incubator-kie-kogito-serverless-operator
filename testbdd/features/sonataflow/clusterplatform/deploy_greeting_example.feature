Feature: Deploy SonataFlow Operator and SonataFlowClusterPlatform and SonataFlowPlatform and greeting example in separate namespace in preview mode

  @previewMode
  Scenario: order-processing DevMode E2E test
    Given Namespace is created
    When SonataFlow Operator is deployed
    When SonataFlowClusterPlatform is deployed