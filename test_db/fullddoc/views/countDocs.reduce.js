// could be replaced by _stats
function(keys, values, rereduce) {
  if (rereduce) {
    return {
      'sum': values.reduce(function(a, b) { return a + b.sum }, 0),
      'min': values.reduce(function(a, b) { return Math.min(a, b.min) }, Infinity),
      'max': values.reduce(function(a, b) { return Math.max(a, b.max) }, -Infinity),
      'count': values.reduce(function(a, b) { return a + b.count }, 0),
      'sumsqr': values.reduce(function(a, b) { return a + b.sumsqr }, 0)
    }
  } else {
    return {
      'sum': sum(values),
      'min': Math.min.apply(null, values),
      'max': Math.max.apply(null, values),
      'count': values.length,
      'sumsqr': (function() {
        var sumsqr = 0;

        values.forEach(function (value) {
          sumsqr += value * value;
        });

        return sumsqr;
      })(),
    }
  }
}