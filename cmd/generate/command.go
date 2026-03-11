package generate

import "github.com/spf13/cobra"

var (
	transactionalModule bool
	useMongoPersistence bool
	useGormPersistence  bool
	inMain              bool
	repositoryDB        string
)

var executeFn = execute

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate [type] [name]",
		Aliases: []string{"g"},
		Short:   "Generate Keel components",
		Args:    cobra.ExactArgs(2),
		RunE:    runGenerate,
	}

	cmd.Flags().BoolVar(&transactionalModule, "transactional", false, "Generate module without controllers (transaction/background module)")
	cmd.Flags().BoolVar(&useMongoPersistence, "mongo", false, "Generate a Mongo-backed repository for the module or repository")
	cmd.Flags().BoolVar(&useGormPersistence, "gorm", false, "Generate a GORM-backed repository for the module or repository")
	cmd.Flags().BoolVar(&inMain, "in-main", false, "For standalone controller: generate routes directly in cmd/main.go")
	cmd.Flags().StringVar(&repositoryDB, "repository-db", "", "Repository backend to use: gorm or mongo (auto-detected/prompted when omitted)")

	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	opts := Options{
		TransactionalModule: transactionalModule,
		UseMongoPersistence: useMongoPersistence,
		UseGormPersistence:  useGormPersistence,
		ControllerInMain:    inMain,
		RepositoryBackend:   repositoryDB,
	}
	return executeFn(args[0], args[1], opts)
}
