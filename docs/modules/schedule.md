# x/schedule — Process Scheduling

## Purpose

Schedule contract/process execution at future block heights. Supports creation with prepaid fees, pause/resume, cancellation with refund, and fee top-up.

## CLI Commands

### Transactions
```bash
legbled tx schedule create-schedule [target-block] [payload] --fee 100000ulgm --from alice
legbled tx schedule pause-schedule [process-id] --from alice
legbled tx schedule resume-schedule [process-id] --from alice
legbled tx schedule cancel-schedule [process-id] --from alice
legbled tx schedule add-fee [process-id] --amount 50000ulgm --from alice
```

### Queries
```bash
legbled query schedule params
legbled query schedule process [id]
legbled query schedule processes-by-creator [address]
```

## Events

`ScheduleCreated`, `SchedulePaused`, `ScheduleResumed`, `ScheduleCanceled`, `FeeAdded`

## Dependencies

BankKeeper, BVMKeeper (execution)

## EndBlock

Processes due schedules and cleans up expired processes.
