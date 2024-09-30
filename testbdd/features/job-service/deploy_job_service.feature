Feature: Deploy SonataFlow Operator and SonataFlowPlatform with Job Service

  @Smoke
  Scenario: Verify Job Service deployment
    Given Namespace is created
    When SonataFlow Operator is deployed
    When SonataFlowPlatform with JobService is deployed