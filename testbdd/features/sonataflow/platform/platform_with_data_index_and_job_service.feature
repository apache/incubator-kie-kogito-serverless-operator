Feature: Deploy SonataFlowPlatform with Data Index and JobService using Postgres database

  Background:
    Given Namespace is created
    When SonataFlow Operator is deployed
    When Postgres is deployed
    When SonataFlowPlatform with DataIndexAndJobService using Postgres is deployed

  @devMode
  Scenario: Deploy order-processing example in dev mode and verify its functionality
    When SonataFlow orderprocessing example is deployed
    Then SonataFlow "orderprocessing" has the condition "Running" set to "True" within 5 minutes
    Then SonataFlow "orderprocessing" is addressable within 1 minute
    Then HTTP POST request as Cloud Event on SonataFlow "orderprocessing" is successful within 1 minute with path "", headers "content-type= application/json,ce-specversion= 1.0,ce-source= /from/localhost,ce-type= orderEvent,ce-id= f0643c68-609c-48aa-a820-5df423fa4fe0" and body:
    """json
    {"id":"f0643c68-609c-48aa-a820-5df423fa4fe0",
    "country":"Czech Republic",
    "total":10000,
    "description":"iPhone 12"
    }
    """

    Then Deployment "event-listener" pods log contains text 'source: /process/shippinghandling' within 1 minutes
    Then Deployment "event-listener" pods log contains text 'source: /process/fraudhandling' within 1 minutes
    Then Deployment "event-listener" pods log contains text '"id": "f0643c68-609c-48aa-a820-5df423fa4fe0",' within 1 minutes
    Then Deployment "event-listener" pods log contains text '"country": "Czech Republic",' within 1 minutes
    Then Deployment "event-listener" pods log contains text '"total": 10000,' within 1 minutes
    Then Deployment "event-listener" pods log contains text '"description": "iPhone 12",' within 1 minutes
    Then Deployment "event-listener" pods log contains text '"fraudEvaluation": true' within 1 minutes
    Then Deployment "event-listener" pods log contains text '"shipping": "international"' within 1 minutes
    Then SonataFlow "order-processing" pods log does not contain text 'ERROR' within 0 minutes

  @devMode
  Scenario: Deploy greeting-example in dev mode and verify its functionality
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

  @previewMode
  Scenario: Deploy callbackstate-timeouts example in dev mode and verify its functionality
    When SonataFlow callbackstatetimeouts example is deployed
    Then SonataFlow "callbackstatetimeouts" has the condition "Running" set to "True" within 5 minutes
    Then SonataFlow "callbackstatetimeouts" is addressable within 1 minute
    Then HTTP GET request on SonataFlow "callbackstatetimeouts" is successful within 1 minute with path "callbackstatetimeouts", expectedResponseContains ''
