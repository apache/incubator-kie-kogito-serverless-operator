Feature: Deploy SonataFlow Operator and SonataFlowPlatform with greeting example in dev mode and verify the functionality

  @devMode1
  Scenario: greeting-example in DevMode E2E test
    Given Namespace is created
    When SonataFlow Operator is deployed
    When SonataFlowPlatform is deployed
    When SonataFlow greeting example is deployed
    Then SonataFlow "greeting" has the condition "Running" set to "True" within 5 minutes
    Then SonataFlow "greeting" is addressable within 1 minute
    Then HTTP POST request on SonataFlow "greeting" is successful within 1 minute with path "greeting", expectedResponseContains '"workflowdata":{"name":"Petr Pavel","language":"Spanish","greeting":"Saludos desde JSON Workflow, "}' and body:
    """json
    {"name":"Petr Pavel",
    "language":"Spanish"
    }
    """

    Then HTTP GET request on SonataFlow "greeting" is successful within 1 minute with path "greeting", expectedResponseContains ''
    Then SonataFlow "greeting" pods log contains text '"name" : "Petr Pavel",' within 1 minutes
    Then SonataFlow "greeting" pods log contains text '"language" : "Spanish"' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'Starting workflow' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'Start' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'ChooseOnLanguage' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'GreetInSpanish' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'Join-GreetPerson' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'GreetPerson' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'Saludos desde JSON Workflow' within 1 minutes
    Then SonataFlow "greeting" pods log contains text 'End' within 1 minutes
    Then SonataFlow "greeting" pods log does not contain text 'ERROR' within 0 minutes