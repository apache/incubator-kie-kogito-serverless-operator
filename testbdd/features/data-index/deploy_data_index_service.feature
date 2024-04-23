Feature: Deploy SonataFlow Operator and SonataFlowPlatform with Data Index Service

  @Smoke  
  Scenario: Verify Data Index Service deployment
    Given Namespace is created
    When SonataFlow Operator is deployed
    When SonataFlowPlatform with DataIndexService is deployed