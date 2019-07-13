## Information about the Config Files:
All evaluation config files need to be placed in this folder (./).
There are two type of config files: 
- The Creater Config: Specifies parameters which are used by the configGenerator to generate a variety of Workflow Configs. They are used for generate several configurations of the recommender workflow at once
- The Workflow Config for testing: Specifies the parameter for the workflow inside the Recommender.

### Structure of the Creater Config:
`{
    "Conds": ["tooFewRecommendations","aboveThreshold","tooUnlikelyRecommendationsCondition"],
    "Merger": ["max","avg"],
    "Splitter": ["everySecondItem", "twoSupportRanges"], 
    "Steps":["stepsizeLinear","stepsizeProportional"],
    "MaxThreshold": 3, 
    "MaxParallel": 4, 
    "MaxFloat": 0 
}`

`Conds`: Condition we want to include in the test config files

`Merger`: Merger strats for the splitPropertyBackoff we want to includ

`Splitter`: Splitter strats for the splitPropertyBackoff we want to include

`Steps`: Stepsizefunction for the deleteLowFrequency Backoff Strat

`MaxThreshold`: maximal Threshold for the condition Conds

`MaxParallel`: maximal Threshold for parallel executions in the deleteLowFrequency Backoff

`MaxFloat`: maximal threshold for too unlikely matches condition


### Structure of the Workflow Config File for Evaluation:
`{
    "Testset":"../testdata/10M.nt_1in2_test.gz"
    "Layers":[ 
    {
            "Condition":"tooFewRecommendations", 
            "Backoff":"deleteLowFrequency",
            "Threshold":1, 
            "ThresholdFloat":0.09, 
            "Merger":"", 
            "Splitter":"",
            "Stepsize":"stepsizeLinear",
            "ParallelExecutions":1 
        },{
            "Condition":"always", 
            "Backoff":"standard",
            "Threshold":0,
            "ThresholdFloat":0,
            "Merger":"",
            "Splitter":"",
            "Stepsize":"",
            "ParallelExecutions":0
        }
    ]
}`

`Layers`: Layers for the workflow. First element of the list is executed first inside the workflow. Each layer specifies condition for enablement, the backoff strategy, and condition and backoff specific parameter:

`Testset`: Testset to use for evaluation

`Condition`: Condition for enablement of that layer

`Backoff`: backoff strategy that fires if enabled

`Threshold`: threshold for the condition tooFewRecommendations,aboveThreshold

`ThresholdFolat`: threshhold for tooUnlikelyRecommendations condition

`Merger`: Merger strategy for splitProperty Backoff

`Splitter`: Splitter strategy for splitProperty Backoff

`Stepsize`: Stepsize function for deleteLowFrequency Backoff

`ParallelExecutions` Number of parallel executions in the deleteLow frequency backoff

