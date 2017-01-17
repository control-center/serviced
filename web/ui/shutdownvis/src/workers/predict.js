/**
 * leastsquares calculates the slope and y-intercept of the line of best fit,
 * given an array of tuples of numbers. Returns a function that will produce
 * y for a given x.
 */
function leastSquares(series) {
  // series is an array of tuples [ [ts, val]... ]

  // Bomb out if we don't have enough to draw a line
  if (series.length < 2) {
    return x => NaN;
  }

  // Unzip the series into x and y arrays
  let zipped = series.map((_,c) => series.map(row => row[c]));

  // Get the mean of x and mean of y
  let [xmean, ymean] = zipped.map(a => a.reduce((x,y) => x+y)/a.length);

  // Calculate the sum of x*x and x*y, normalizing around the origin by
  // subtracting the mean
  let sumxx = zipped[0].reduce((a,b)=> a + Math.pow(b - xmean, 2), 0);
  let sumxy = series.reduce((a,[x,y]) => a + (x-xmean)*(y-ymean), 0);

  // Calculate the slope and y-intercept of the line of best fit
  let m = sumxy/sumxx;
  let b = ymean - m * xmean;

  // Return a function that will produce y for a given x
  return x => m*x+b;
}

// dataWindow returns the slice of the series that includes the index and the
// previous thewindow seconds.
function dataWindow(series, index, thewindow=30) {
  let ts = series[index][0],
      bound = ts - thewindow;
  for (var idx = index; idx > 0; idx--) {
    if (series[idx][0] <= bound) {
      return series.slice(idx, index + 1);
    }
  }
  // We'll return an empty window rather than partial data, so we don't get junk
  // predictions at the beginning.
  return [];
}

const mean = series => series.reduce(([sumx, sumy], [x, y]) => [sumx+x, sumy+y], [0, 0]).map(a => a/series.length);

// A faster version that calculates least squares while iterating across
// a series, rather than recalculating a window for each point independently.
function fastProjectSeries(series, thewindow=5*60*1000, lookahead=20*60*1000) {
  let currentWindow = [];
  let [meanx, meany] = mean(series);
  let sumxx = 0, sumxy = 0;

  let projected = [];

  while (series.length) {
    // Add the next datapoint to the window
    let [curx, cury] = series.shift();
    currentWindow.push([curx, cury]);

    // Update our running sum of squares
    sumxx += Math.pow(curx-meanx, 2);
    sumxy += (curx-meanx)*(cury-meany);

    // Reconcile the boundaries until we have a valid window
    while (currentWindow[0][0] < currentWindow[currentWindow.length-1][0]-thewindow) {
      // Pop one off up in here
      let [remx, remy] = currentWindow.shift();
      // Subtract the gone-ass data from our running sum of squares
      sumxx -= Math.pow((remx-meanx), 2);
      sumxy -= (remx-meanx)*(remy-meany);
    }

    if (currentWindow.length < 2) {
      continue;
    }

    let m = sumxy/sumxx;
    let b = meany - m * meanx;
    let futurex = curx + lookahead;

    projected.push([curx, futurex*m+b]);
  }

  return projected;
}

onmessage = function(e){
    let {data, theWindow, lookAhead} = e.data;
    let result = fastProjectSeries(data, theWindow, lookAhead);
    postMessage(result);
};
