package codegenerator

import (
	"github.com/Mario-Jimenez/gocompiler/parser"
)

/*
	additionExpression:
		multiplicationExpression (
			additionFactor multiplicationExpression
		)* # additionTree
		;
*/

func (v *visitor) VisitAdditionTree(ctx *parser.AdditionTreeContext) interface{} {
	v.Visit(ctx.MultiplicationExpression(0))

	totalBranches := len(ctx.AllMultiplicationExpression())
	index := 1
	for index < totalBranches {
		v.Visit(ctx.MultiplicationExpression(index))

		v.Visit(ctx.AdditionFactor(index - 1))

		index++
	}

	return nil
}
