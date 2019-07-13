Both backoff strategies fight the problem that too many given properties in the
request result in no possibile recommendation, since inside the schematree no candiate
node can be found that has all the properties from the request inside its prefix.

## Delete Low Frequency Backoff

1) Sort incoming request according to frequency 
2) Create subsets by deleting  p  times the i less frequent items, i variating according to the stepsize function
3) Run p recommenders in parallel
4) choose that one which satisfies the condition and deleted less number of properties

LinearStepsize: take away the least frequent property one after another (#parallelExecution times)

ProportionalStepsize: determine the least 40% of incoming property set. take away (1/#parallelExecutions) one after another.

## Split Property Set 

1) Sort incoming properties according to their frequency (which we get from the tree)
2) Split the property set into two subsets 
3) Perform recommendation on both subsets (in parallel which works brilliantly in go)
4) Merge (avg or max)
	
Split strategies are:

EverySecondItem: The result are two more or less equally distrbuted sets in terms of property frequency

TwoFreqeuncyRanges: The result are two sets, one containing all the high frequent properties, the other all the lows.