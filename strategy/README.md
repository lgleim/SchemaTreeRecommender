# Strategy Module

Strategy is responsible of executing the correct recomendation procedure given an initial input assessment.

The main element, the Workflow, is an ordered list of (Condition,Procedure) pairs. It will run through
all entries in order, and the first Condition that holds true will trigger the execution of the Procedure.
This Procedure produces the final recommendation and only a single Procedure will be triggered per request.

It is possible to customize their own strategy via code, or use one of the preset strategies.
