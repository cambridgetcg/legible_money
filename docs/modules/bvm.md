# x/bvm — Belief Virtual Machine

## Purpose

Smart contract execution environment with bytecode interpreter, opcodes, and scheduled execution. Contracts can be deployed, called with value transfer, and scheduled for future block execution. Gas-bridged between SDK and VM. Capability propagation to sub-calls.

## Key Concepts

- **Bytecode Interpreter**: Stack-based VM with 100+ opcodes
- **Host Functions**: Bridge between VM and chain state (knowledge queries, staking info)
- **Gas Bridge**: SDK gas consumed proportionally to VM gas
- **Scheduled Execution**: Contracts can be triggered at future blocks (5M gas budget per block)
- **Panic Recovery**: All execution paths recover from panics
- **Memory Safety**: Offset overflow protection on 15 opcode sites

## CLI Commands

### Transactions
```bash
legbled tx bvm deploy-contract [bytecode-hex] --constructor-args [hex] --deposit 100000ulgm --from alice
legbled tx bvm call-contract [address] [input-hex] --value 50000ulgm --gas-limit 1000000 --from bob
legbled tx bvm schedule-contract [address] [input-hex] --execute-at-block 100000 --from alice
legbled tx bvm cancel-schedule [schedule-id] --from alice
```

### Queries
```bash
legbled query bvm params
legbled query bvm contract [address]
legbled query bvm contracts-by-creator [creator]
legbled query bvm code [address]
legbled query bvm contract-state [address] [key-hex]
legbled query bvm schedule [id]
```

## Parameters

Module params for gas limits, execution costs, max contract size, scheduled budget per block.

## Events

`ContractDeployed`, `ContractExecuted`, `ScheduleCreated`, `ScheduleExecuted`

## Dependencies

BankKeeper, AccountKeeper, ScheduleKeeper

## BeginBlock

Executes pending scheduled contracts up to 5M gas budget.
