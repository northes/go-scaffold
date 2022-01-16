package commandx

import (
	"fmt"
	"github.com/spf13/cobra"
)

// CommandLine 命令行
type CommandLine struct {
	cmd *cobra.Command
}

// NewCommandLine 构造函数
func NewCommandLine(cmd *cobra.Command) *CommandLine {
	return &CommandLine{
		cmd: cmd,
	}
}

// Registry 注册命令行实体
func (c CommandLine) Registry(commands []*Command) {
	entities := make([]*cobra.Command, 0, len(commands))

	for _, cs := range commands {
		childrenEntities := make([]*cobra.Command, 0, len(cs.Children))

		for _, csc := range cs.Children {
			if csc.OptionFunc != nil {
				csc.OptionFunc(csc.Entity)
			}
			childrenEntities = append(childrenEntities, csc.Entity)
		}

		if cs.OptionFunc != nil {
			cs.OptionFunc(cs.Entity)
		}
		cs.Entity.AddCommand(childrenEntities...)

		entities = append(entities, cs.Entity)
	}

	c.cmd.AddCommand(entities...)
}

// RegistryBusiness 注册业务的命令行实体
func (c CommandLine) RegistryBusiness(commands []*Command) {
	cmd := []*Command{
		{
			Entity: &cobra.Command{
				Use:   "business",
				Short: "业务命令",
				Long:  "business 仅作为运行业务的一个入口方式（注意：不应该在 cron 中频繁运行，这会造成性能问题）",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println(cmd.UsageString())
				},
			},
			Children: commands,
		},
	}

	c.Registry(cmd)
}

// RegistryScript 注册临时脚本的命令行实体
func (c CommandLine) RegistryScript(commands []*Command) {
	cmd := []*Command{
		{
			Entity: &cobra.Command{
				Use:   "script",
				Short: "临时脚本命令",
				Long:  "script 仅运行临时性的任务（注意：不应该在 cron 中频繁运行，这会造成性能问题）",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println(cmd.UsageString())
				},
			},
			Children: commands,
		},
	}

	c.Registry(cmd)
}

// Command 命令实体
// 对 cobra.Command 的扩展
type Command struct {
	// Entity cobra.Command 命令行实体
	Entity *cobra.Command

	// OptionFunc cobra.Command 的选项设置函数
	OptionFunc func(*cobra.Command)

	// Children 子命令
	Children []*Command
}
