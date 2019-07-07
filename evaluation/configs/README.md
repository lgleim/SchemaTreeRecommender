## Information about the Config Files:
All config files need to be placed in this folder (./).
There are two type of config files: 
- The Creater Config: Specifies parameters which are used by the configGenerator to generate a variety of Workflow Configs. They are used for generate several configurations of the recommender workflow at once
- The Workflow Config: Specifies the parameter for the workflow inside the Recommender.

### Structure of the Creater Config:
`{
    "Conds": ["tooFewRecommendations","aboveThreshold","tooUnlikelyRecommendationsCondition"], //Condition we want to include in the test config files
    "Merger": ["max","avg"], // Merger strats for the splitPropertyBackoff we want to include
    "Splitter": ["everySecondItem", "twoSupportRanges"], // Splitter strats for the splitPropertyBackoff we want to include
    "Steps":["stepsizeLinear","stepsizeProportional"], // Stepsizefunction for the deleteLowFrequency Backoff Strat
    "MaxThreshold": 3, // maximal Threshold for the condition Conds
    "MaxParallel": 4, // maximal Threshold for parallel executions in the deleteLowFrequency Backoff
    "MaxFloat": 0 // maximal threshold for too unlikely matches condition
}`

### Structure of the Workflow Config File:
`{
    "Testset":"../testdata/10M.nt_1in2_test.gz", // testset to use for evaluation
    "Layers":[ // Layers for the workflow. First element of the list is executed first inside the workflow. Each layer specifies condition for enablement, the backoff strategy, and condition and backoff specific parameter:
        {
            "Condition":"tooFewRecommendations", // Condition for enablement of that layer
            "Backoff":"deleteLowFrequency", // backoffstrategy that fires if enabled
            "Threshold":1, // threshold for the condition tooFewRecommendations,aboveThreshold
            "ThresholdFloat":0.09, // threshhold for tooUnlikelyRecommendations condition
            "Merger":"", // Merger strategy for splitProperty Backoff
            "Splitter":"", // Splitter strategy for splitProperty Backoff
            "Stepsize":"stepsizeLinear", // Stepsize function for deleteLowFrequency Backoff
            "ParallelExecutions":1 // Number of parallel executions in the deleteLow frequency backoff
        },{
            "Condition":"always", // Final condition should always fire and execute the standard recommender
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
