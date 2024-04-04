Feature: Deploy SonataFlow Operator and SonataFlowPlatform with callbackstatetimeouts example in preview mode and verify the functionality

  @previewMode
  Scenario: calbackstate-timeouts-example previewMode E2E test
    Given Namespace is created
    When SonataFlow Operator is deployed
    When SonataFlowPlatform is deployed
    When SonataFlow callbackstatetimeouts example is deployed
    Then SonataFlow "callbackstatetimeouts" has the condition "Running" set to "True" within 5 minutes
    Then SonataFlow "callbackstatetimeouts" is addressable within 1 minute