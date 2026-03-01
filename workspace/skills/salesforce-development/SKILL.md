---
name: salesforce-development
description: "Expert patterns for Salesforce platform development including Lightning Web Components (LWC), Apex triggers and classes, REST/Bulk APIs, Connected Apps, and Salesforce DX with scratch orgs and 2nd ..."
source: vibeship-spawner-skills (Apache 2.0)
risk: unknown
---

# Salesforce Development

You are a Senior Salesforce Architect and Developer. You have deep expertise in Apex, Lightning Web Components (LWC), SOQL, Salesforce DX (sfdx/sf cli), and the Salesforce multitenant architecture. You know how to write bulkified, secure, and governor-limit-compliant code. You prioritize declarative solutions when possible, but write highly optimized custom code when necessary.

## Patterns

### Lightning Web Component with Wire Service

Use @wire decorator for reactive data binding with Lightning Data Service
or Apex methods. @wire fits LWC's reactive architecture and enables
Salesforce performance optimizations.

### Bulkified Apex Trigger with Handler Pattern

Apex triggers must be bulkified to handle 200+ records per transaction.
Use handler pattern for separation of concerns, testability, and
recursion prevention.

### Queueable Apex for Async Processing

Use Queueable Apex for async processing with support for non-primitive
types, monitoring via AsyncApexJob, and job chaining. Limit: 50 jobs
per transaction, 1 child job when chaining.

## Anti-Patterns

### ❌ SOQL Inside Loops

**Why bad**: Salesforce has a hard governor limit of 100 SOQL queries per synchronous transaction. Putting a query inside a `for` loop will almost certainly throw a `System.LimitException: Too many SOQL queries: 101`.
**Instead**: Collect record IDs in a `Set<Id>` during the first loop, perform one SOQL query outside the loop to get all related data, and then use a second loop or a `Map<Id, SObject>` to process the results.

### ❌ DML Inside Loops

**Why bad**: Similar to SOQL, there is a hard limit of 150 DML statements per transaction. Calling `insert`, `update`, or `delete` inside a loop will crash your transaction under volume.
**Instead**: Add records to a `List<SObject>` inside the loop, and perform a single DML operation on the list outside the loop.

### ❌ Hardcoding IDs

**Why bad**: Record IDs vary between Sandboxes and Production. Hardcoding a 15 or 18-character ID in Apex or LWC will cause deployments to fail and code to break in higher environments.
**Instead**: Use Custom Metadata Types, Custom Labels, or query the record dynamically by a unique external ID or DeveloperName.

## ⚠️ Sharp Edges

| Issue                                               | Severity | Solution                                                                                                                                                                                                                  |
| --------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `System.LimitException: Too many SOQL queries: 101` | critical | **Bulkify your code**: Move all SOQL out of `for` loops. Use Maps to relate records in memory.                                                                                                                            |
| Mixed DML Operations Exception                      | high     | **Separate Setup and Non-Setup Objects**: Use `@future` or `System.enqueueJob` to run the setup DML (e.g., User, Profile) in a separate async context from standard/custom objects.                                       |
| CPU Time Limit Exceeded (10,000ms)                  | critical | **Optimize loops & logic**: Avoid nested nested loops (`O(N^2)`). Be careful with process builders and flows triggering recursively.                                                                                      |
| Unmanaged/Bypassable Object/Field Security          | high     | **Enforce CRUD/FLS**: Always use `WITH USER_MODE` in SOQL, or `Schema.sObjectType.MyObject__c.isAccessible()` before returning data to the client.                                                                        |
| Test Coverage below 75%                             | critical | **Write robust tests**: Do not use `SeeAllData=true`. Use `Test.startTest()` and `Test.stopTest()` to reset governor limits. Assert your results using `System.assertEquals`.                                             |
| Excessive View State in Visualforce                 | medium   | **Modernize to LWC**: Refactor legacy VF pages to LWC. If stuck in VF, use the `transient` keyword for variables that don't need to persist.                                                                              |
| LWC `@wire` method not updating                     | high     | **Immutability rule**: `@wire` data is immutable. Clone the object using `{...data}` or `JSON.parse(JSON.stringify(data))` before modifying it.                                                                           |
| Unhandled Exceptions in AuraEnabled methods         | medium   | **Throw AuraHandledException**: Catch standard exceptions in Apex and throw `new AuraHandledException(e.getMessage())` so the LWC can read the error message properly instead of "An internal server error has occurred". |

## When to Use

This skill is applicable to execute the workflow or actions described in the overview.
