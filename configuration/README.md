### Structure of the Workflow Config File:
`{
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

`Condition`: Condition for enablement of that layer

`Backoff`: backoff strategy that fires if enabled

`Threshold`: threshold for the condition tooFewRecommendations,aboveThreshold

`ThresholdFolat`: threshhold for tooUnlikelyRecommendations condition

`Merger`: Merger strategy for splitProperty Backoff

`Splitter`: Splitter strategy for splitProperty Backoff

`Stepsize`: Stepsize function for deleteLowFrequency Backoff

`ParallelExecutions` Number of parallel executions in the deleteLow frequency backoff

The difference to a workflow config file in the evaluation is the missing testset field.