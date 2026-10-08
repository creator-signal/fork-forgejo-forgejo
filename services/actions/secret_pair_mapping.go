// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"strings"

	secret_model "forgejo.org/models/secret"

	"github.com/rhysd/actionlint"
)

// validatePairWorkflowCallMapping uses the existing expression grammar. Only
// secrets: inherit may carry the pair; explicit names/references cannot split it.
func validatePairWorkflowCallMapping(mapping map[string]string, callerSecrets map[string]string) error {
	_, hasUsername := callerSecrets[secret_model.PairUsernameName]
	_, hasPassword := callerSecrets[secret_model.PairPasswordName]
	pairPresent := hasUsername || hasPassword
	for name, value := range mapping {
		if secret_model.IsManagedSecretName(name) {
			return secret_model.ErrPairConflict
		}
		for {
			start := strings.Index(value, "${{")
			if start < 0 {
				break
			}
			input := value[start+3:]
			// Bound the existing parser before recursive AST construction.
			tokens, consumed, lexErr := actionlint.LexExpression(input[:min(len(input), 65536)])
			if lexErr != nil || consumed <= 0 || consumed > 65536 || len(tokens) > 4096 {
				return secret_model.ErrPairConflict
			}
			depth := 0
			for _, token := range tokens {
				if token.Kind == actionlint.TokenKindLeftParen || token.Kind == actionlint.TokenKindLeftBracket {
					depth++
				}
				if token.Kind == actionlint.TokenKindRightParen || token.Kind == actionlint.TokenKindRightBracket {
					depth--
				}
				if depth < 0 || depth > 64 {
					return secret_model.ErrPairConflict
				}
			}
			lexer := actionlint.NewExprLexer(input[:consumed])
			root, err := actionlint.NewExprParser().Parse(lexer)
			if err != nil || lexer.Err() != nil {
				return secret_model.ErrPairConflict
			}
			denied := false
			actionlint.VisitExprNode(root, func(node, parent actionlint.ExprNode, entering bool) {
				if !entering {
					return
				}
				variable, ok := node.(*actionlint.VariableNode)
				if !ok || !strings.EqualFold(variable.Name, "secrets") {
					return
				}
				switch access := parent.(type) {
				case *actionlint.ObjectDerefNode:
					if access.Receiver != node || secret_model.IsManagedSecretName(access.Property) {
						denied = true
					}
				case *actionlint.IndexAccessNode:
					key, static := access.Index.(*actionlint.StringNode)
					if access.Operand != node || !static {
						if pairPresent {
							denied = true
						}
					} else if secret_model.IsManagedSecretName(key.Value) { denied = true }
				default:
					if pairPresent {
						denied = true
					}
				}
			})
			if denied {
				return secret_model.ErrPairConflict
			}
			value = input[consumed:]
		}
	}
	return nil
}
