package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/tree/types"
)

// NewTxCmd returns the transaction commands for the tree module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tree module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateProjectCmd(),
		NewAddTaskCmd(),
		NewAssignTaskCmd(),
		NewStartWorkCmd(),
		NewSubmitDeliverableCmd(),
		NewApproveDeliverableCmd(),
		NewRejectDeliverableCmd(),
		NewDeployServiceCmd(),
		NewCallServiceCmd(),
		NewCreateSeedCmd(),
	)

	return txCmd
}

// NewCreateProjectCmd creates a CLI command for MsgCreateProject.
func NewCreateProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-project [title] [domain] [description]",
		Short: "Create a new project in the knowledge tree",
		Long: `Create a new project with a title, domain, and description.

The creator is derived from the --from flag. Optionally specify a budget
and required capabilities for the project.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			budget, _ := cmd.Flags().GetString("budget")
			caps, _ := cmd.Flags().GetStringSlice("capabilities")

			msg := &types.MsgCreateProject{
				Creator:              clientCtx.GetFromAddress().String(),
				Title:                args[0],
				Domain:               args[1],
				Description:          args[2],
				Budget:               budget,
				RequiredCapabilities: caps,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("budget", "", "Initial budget for the project (e.g. 1000000ulgm)")
	cmd.Flags().StringSlice("capabilities", nil, "Required capabilities (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewAddTaskCmd creates a CLI command for MsgAddTask.
func NewAddTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-task [project-id] [title] [description]",
		Short: "Add a task to a project",
		Long: `Add a new task to an existing project. The task creator is derived
from the --from flag. Optionally specify a bounty amount and required
capabilities for the task.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bounty, _ := cmd.Flags().GetString("bounty")
			caps, _ := cmd.Flags().GetStringSlice("capabilities")

			msg := &types.MsgAddTask{
				Creator:              clientCtx.GetFromAddress().String(),
				ProjectId:            args[0],
				Title:                args[1],
				Description:          args[2],
				Bounty:               bounty,
				RequiredCapabilities: caps,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("bounty", "", "Bounty amount for the task (e.g. 100000ulgm)")
	cmd.Flags().StringSlice("capabilities", nil, "Required capabilities (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewAssignTaskCmd creates a CLI command for MsgAssignTask.
func NewAssignTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assign-task [task-id] [assignee-address]",
		Short: "Assign a task to an agent",
		Long: `Assign an open task to a specific agent address. The assigner
is derived from the --from flag and must have authority over the project.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAssignTask{
				Assigner: clientCtx.GetFromAddress().String(),
				TaskId:   args[0],
				Assignee: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewStartWorkCmd creates a CLI command for MsgStartWork.
func NewStartWorkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-work [task-id]",
		Short: "Start working on an assigned task",
		Long: `Signal that you are beginning work on a task. The worker address
is derived from the --from flag and must match the task assignee.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgStartWork{
				Worker: clientCtx.GetFromAddress().String(),
				TaskId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitDeliverableCmd creates a CLI command for MsgSubmitDeliverable.
func NewSubmitDeliverableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-deliverable [task-id] [deliverable-hash]",
		Short: "Submit a deliverable for a task",
		Long: `Submit work output for review. The worker is derived from the --from flag.
Provide the content hash of the deliverable and optionally a URI where the
deliverable can be accessed.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			uri, _ := cmd.Flags().GetString("uri")

			msg := &types.MsgSubmitDeliverable{
				Worker:          clientCtx.GetFromAddress().String(),
				TaskId:          args[0],
				DeliverableHash: args[1],
				DeliverableUri:  uri,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("uri", "", "URI where the deliverable can be accessed")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewApproveDeliverableCmd creates a CLI command for MsgApproveDeliverable.
func NewApproveDeliverableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve-deliverable [task-id]",
		Short: "Approve a submitted deliverable",
		Long: `Approve the deliverable submitted for a task, marking it complete
and releasing the bounty. The approver is derived from the --from flag
and must have authority over the project.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgApproveDeliverable{
				Approver: clientCtx.GetFromAddress().String(),
				TaskId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRejectDeliverableCmd creates a CLI command for MsgRejectDeliverable.
func NewRejectDeliverableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject-deliverable [task-id] [reason]",
		Short: "Reject a submitted deliverable",
		Long: `Reject the deliverable submitted for a task with a reason. The rejector
is derived from the --from flag and must have authority over the project.
After max rejections, the task enters disputed status.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRejectDeliverable{
				Rejector: clientCtx.GetFromAddress().String(),
				TaskId:   args[0],
				Reason:   args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeployServiceCmd creates a CLI command for MsgDeployService.
func NewDeployServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy-service [name] [endpoint]",
		Short: "Deploy a service to the tree marketplace",
		Long: `Deploy a new service with a name and endpoint URL. The deployer is
derived from the --from flag. Optionally set a description, per-call price,
subscription price, and capability tags.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			pricePerCall, _ := cmd.Flags().GetString("price-per-call")
			subscriptionPrice, _ := cmd.Flags().GetString("subscription-price")
			caps, _ := cmd.Flags().GetStringSlice("capabilities")

			msg := &types.MsgDeployService{
				Deployer:          clientCtx.GetFromAddress().String(),
				Name:              args[0],
				Endpoint:          args[1],
				Description:       description,
				PricePerCall:      pricePerCall,
				SubscriptionPrice: subscriptionPrice,
				Capabilities:      caps,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "Service description")
	cmd.Flags().String("price-per-call", "", "Price per service call (e.g. 1000ulgm)")
	cmd.Flags().String("subscription-price", "", "Subscription price per period (e.g. 100000ulgm)")
	cmd.Flags().StringSlice("capabilities", nil, "Service capabilities (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewCallServiceCmd creates a CLI command for MsgCallService.
func NewCallServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call-service [service-id] [payload]",
		Short: "Call a deployed service",
		Long: `Invoke a deployed service with a payload. The caller is derived from
the --from flag. Specify a maximum fee you are willing to pay.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			maxFee, _ := cmd.Flags().GetString("max-fee")

			msg := &types.MsgCallService{
				Caller:    clientCtx.GetFromAddress().String(),
				ServiceId: args[0],
				Payload:   []byte(args[1]),
				MaxFee:    maxFee,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("max-fee", "", "Maximum fee willing to pay (e.g. 5000ulgm)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewCreateSeedCmd creates a CLI command for MsgDetectOpportunity.
func NewCreateSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-seed [domain] [description]",
		Short: "Detect a new opportunity seed in a knowledge domain",
		Long: `Submit an opportunity seed detected in a knowledge domain. The detector
is derived from the --from flag. Optionally reference related fact IDs
that support the opportunity.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			relatedFacts, _ := cmd.Flags().GetStringSlice("related-facts")

			msg := &types.MsgDetectOpportunity{
				Detector:     clientCtx.GetFromAddress().String(),
				Domain:       args[0],
				Description:  args[1],
				RelatedFacts: relatedFacts,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("related-facts", nil, "Related fact IDs supporting the opportunity (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
