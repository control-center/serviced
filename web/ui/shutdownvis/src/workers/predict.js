const mean = series => series.reduce(([sumx, sumy], [x, y]) => [sumx+x, sumy+y], [0, 0]).map(a => a/series.length);

// At each point in a time series, project the y-value {lookahead} ms in the
// future using a linear regression across a past window of {thewindow} ms.
function projectSeries(series, thewindow=5*60*1000, lookahead=20*60*1000) {
  let currentWindow = [];
  let [meanx, meany] = mean(series);
  let sumx = 0, sumy = 0;
  let sumxx = 0, sumxy = 0;

  let projected = [];

  while (series.length) {
    // Add the next datapoint to the window
    let [curx, cury] = series.shift();
    currentWindow.push([curx, cury]);

    // Update our running sums
    let curadjx = curx-meanx;
    let curadjy = cury-meany;
    sumxx += Math.pow(curadjx, 2);
    sumxy += curadjx*curadjy;
    sumx += curadjx;
    sumy += curadjy;

    // Reconcile the boundaries until we have a valid window
    while (currentWindow[0][0] < curx-thewindow) {
      // Pop one off up in here
      let [remx, remy] = currentWindow.shift();

      // Subtract the gone-ass data from our running sums
      let adjx = remx-meanx;
      let adjy = remy-meany;
      sumxx -= Math.pow(adjx, 2);
      sumxy -= adjx*adjy;
      sumx -= adjx;
      sumy -= adjy;
    }

    let L = currentWindow.length;

    if (L < 2) {
      continue;
    }

    let m = (L*sumxy - sumx*sumy) / (L*sumxx - sumx*sumx);
    let b = ((sumy/L)+meany) - (m * (sumx/L+meanx));
    let futurex = curx + lookahead;

    projected.push([curx, m*futurex+b]);
  }

  return projected;
}

onmessage = function(e){
    let {data, theWindow, lookAhead} = e.data;
    let result = projectSeries(data, theWindow, lookAhead);
    postMessage(result);
};
