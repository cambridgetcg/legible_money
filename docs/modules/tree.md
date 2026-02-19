# x/tree — Projects, Tasks & Services

## Purpose

Work/service marketplace. Tracks projects with tasks, service deployments, and opportunity seeds. Enables delegation of work with deliverable approval flow and bounty payments.

## CLI Commands

### Transactions
```bash
legbled tx tree create-project [title] --budget 10000000ulgm --from founder
legbled tx tree add-task [project-id] [title] --bounty 500000ulgm --from founder
legbled tx tree assign-task [task-id] [worker] --from founder
legbled tx tree start-work [task-id] --from worker
legbled tx tree submit-deliverable [task-id] [content-hash] --from worker
legbled tx tree approve-deliverable [task-id] --from founder
legbled tx tree reject-deliverable [task-id] --reason "..." --from founder
legbled tx tree deploy-service [name] [endpoint] --from provider
legbled tx tree call-service [service-id] [payload] --from user
legbled tx tree create-seed [domain] [description] --from alice
```

### Queries
```bash
legbled query tree params
legbled query tree project [id]
legbled query tree projects-by-founder [address]
legbled query tree task [id]
legbled query tree tasks-by-project [project-id]
legbled query tree service [id]
legbled query tree seed [id]
```

## Events

`ProjectCreated`, `TaskAdded`, `TaskAssigned`, `TaskStarted`, `DeliverableSubmitted`, `DeliverableApproved`, `DeliverableRejected`, `ServiceDeployed`, `ServiceCalled`, `SeedDetected`

## Dependencies

BankKeeper, AccountKeeper

## BeginBlock

Expires seeds past their deadline.
