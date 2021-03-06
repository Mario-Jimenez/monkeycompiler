package handler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/Mario-Jimenez/gocompiler/cli"
	"github.com/Mario-Jimenez/gocompiler/errors"
	"github.com/Mario-Jimenez/gocompiler/identification"
	"github.com/Mario-Jimenez/gocompiler/parser"
	"github.com/Mario-Jimenez/gocompiler/visitor/codegenerator"
	"github.com/Mario-Jimenez/gocompiler/visitor/contextual"
	"github.com/Mario-Jimenez/gocompiler/visitor/graph"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/gin-gonic/gin"
)

// Monkey contains compiler endpoints functions
type Monkey struct{}

// Compile request
type Compile struct {
	Program string `json:"program"`
}

// Compile incoming program
func (*Monkey) Compile(c *gin.Context) {
	// parse incoming request
	var req Compile
	if err := c.ShouldBindJSON(&req); err != nil {
		// response with error, bad request, missing program parameter
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
		return
	}

	// parsing
	parseErrors, parseLines, parseTree := parsing(req.Program)

	// response
	c.JSON(200, gin.H{
		"errors": parseErrors,
		"lines":  parseLines,
		"tree":   parseTree,
	})
}

// parsing program
func parsing(program string) ([]string, []int, interface{}) {
	// compiler input
	input := antlr.NewInputStream(program)

	parserErrors := errors.NewParserErrorListener()

	// compiler lexer
	lexer := parser.NewMonkeyLexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(parserErrors)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// compiler parser
	parser := parser.NewMonkeyParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(parserErrors)

	// initial rule to start parsing process
	tree := parser.Program()

	// tree graph visitor
	graphVisitor := graph.NewVisitor()
	treeGraph := graphVisitor.Visit(tree)

	if parserErrors.Errors() == nil {
		// errors handler for contextual visitor
		contextualErrors := identification.NewErrorsHandler()
		// identification table
		table := identification.NewTable(contextualErrors)
		// contextual analysis visitor
		contextualVisitor := contextual.NewVisitor(table)
		// start of contextual visitor
		contextualVisitor.Visit(tree)

		if contextualErrors.Errors() == nil {
			// code generator visitor
			generatorVisitor := codegenerator.NewVisitor()
			// start of code generator visitor
			generatorVisitor.Visit(tree)

			// save instructions to file
			err := save([]byte(generatorVisitor.Code()))
			if err != nil {
				return []string{
					err.Error(),
				}, []int{}, treeGraph
			}

			return []string{}, []int{}, treeGraph
		}

		return contextualErrors.Errors(), contextualErrors.Lines(), treeGraph
	}

	return parserErrors.Errors(), parserErrors.Lines(), treeGraph
}

// save instructions to file
func save(data []byte) error {
	err := ioutil.WriteFile(cli.InstructionsCode, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed to save file. %s", err.Error())
	}

	return nil
}

// Run program
func (*Monkey) Run(c *gin.Context) {
	// execute code on monkey virtual machine
	result := run()

	// response
	c.JSON(200, gin.H{
		"result": result,
	})
}

// run monkey virtual machine with instructions code
func run() string {
	if _, err := os.Stat(cli.InstructionsCode); err != nil {
		return fmt.Sprintf("%s\n%s\n...failed", "Instructions code not found. Must compile first.", strings.TrimSpace(err.Error()))
	}

	out, err := exec.Command(cli.VM, cli.InstructionsCode).Output()
	if err != nil {
		return fmt.Sprintf("%s\n...failed", strings.TrimSpace(err.Error()))
	}

	return fmt.Sprintf("%s\n...finished", strings.TrimSpace(string(out)))
}
