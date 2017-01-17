import {wrap} from "./utils";

const MARGINS = {top: 10, right: 10, bottom: 100, left: 50};

function applyPaddingToScale(scale, multiplier){
    let [start, end] = scale.domain(),
        range = end - start,
        val = range * multiplier;
    start -= val;
    end += val;
    scale.domain([start, end]);
}

class ThreshyGraph {
    constructor(el){
        // build container with margins
        let svg = d3.select(el);
        let g = svg.append("g")
            .attr("transform", `translate(${MARGINS.left},${MARGINS.top})`);

        this.svg = svg;
        this.g = g;
        this.visG = g.append("g").attr("class", "vis-group");
        this.margins = MARGINS;

        this.data = {};
        this.shutdowns = [];
        this.margin = 0;

        this.calculateSize();
        this.svg.attr("width", this.w).attr("height", this.h);

        this.createScales();
        this.createAxes();

        this.createZoom();
    }

    createZoom(){
        let zoomed = () => {
            let t = d3.event.transform;
            // NOTE - zoom is stateful, so repeatedly applying
            // zoom to scale will go nutso. Always start from 
            // the original scale and apply zoom to it
            this.xScale = t.rescaleX(this._originalXScale);

            // update x axis
            this.xAxis.scale(this.xScale);
            this.g.select("g.axis.axis-x")
                .call(this.xAxis)
                .selectAll(".tick text")
                    .call(wrap, 0);

            // update everyone else
            let {data, margin, shutdowns} = this;
            this.drawLines(data);
            this.drawMargin(margin);
            this.drawShutups(shutdowns);
        };

        this.zoom = d3.zoom()
            .on("zoom", zoomed)
            .scaleExtent([1, 100])
            .translateExtent([[0, 0], [this.w, this.h]]);

        this.createInteractive();
        this.interactive.call(this.zoom);

        this.svg.append("defs").append("clipPath")
            .attr("id", "clip")
            .append("rect")
                .attr("width", this.w)
                .attr("height", this.h);

        this.visG.attr("clip-path", "url(#clip)");
    }

    calculateSize(){
        // calculate size
        let rect = this.svg.node().getBoundingClientRect();
        this.w = rect.width - this.margins.left - this.margins.right;
        this.h = rect.height - this.margins.top - this.margins.bottom;
    }

    createScales(){
        this.xScale = d3.scaleTime().range([0, this.w]);
        this._originalXScale = this.xScale;
        this.yScale = d3.scaleLinear().range([this.h, 0]);
    }

    createAxes(){
        this.yAxis = d3.axisLeft(this.yScale)
            .ticks(0)
            .tickFormat((d,i) => `${Math.floor(d/1000/1000/1000)} GB`);

        this.g.append("g")
            .attr("class", "axis axis-y")
            .call(this.yAxis);

        let timeFormat = d3.timeFormat("%I:%M %p");
        let dateFormat = t => {
            let format = "%b. %d, %Y";
            // TODO - if t is today, return "today"
            // TODO - if t is yesterday, return "yesterday"
            return d3.timeFormat(format)(t);
        };

        this.xAxis = d3.axisBottom(this.xScale)
            .tickFormat(d => `${timeFormat(d)}\n${dateFormat(d)}`)
            .ticks(0)
            .tickPadding(-10);

        this.g.append("g")
            .attr("class", "axis axis-x")
            .attr("clip-path", "url(#clip)")
            .attr("transform", `translate(0,${this.h})`)
            .call(this.xAxis)
        .selectAll(".tick text")
            .call(wrap, 0);
    }

    createInteractive(){
        this.interactive = this.svg.append("rect")
            .attr("class", "interactive")
            .attr("x", 0)
            .attr("y", 0)
            .attr("width", this.w)
            .attr("height", this.h)
            .attr("transform", `translate(${this.margins.left},${this.margins.top})`)
            .style("visibility", "hidden")
            .attr("pointer-events", "all");
    }

    updateAxes(){
        this.yAxis.tickValues(this.yScale.domain());
        this.g.select("g.axis.axis-y")
            .call(this.yAxis);

        this.xAxis.tickValues(this.shutdowns.map(s => s.dp[0]));
        this.g.select("g.axis.axis-x")
            .call(this.xAxis)
        .selectAll(".tick text")
            .call(wrap, 0);
    }


    updateScales(data, margin){
        this._originalXScale.domain([
            d3.min(data, m => d3.min(m.data, d => d[0])),
            d3.max(data, m => d3.max(m.data, d => d[0]))
        ]);
        this.yScale.domain([
            d3.min(data, m => d3.min(m.data, d => d[1])),
            d3.max(data, m => d3.max(m.data, d => d[1]))
        ]);

        // apply margin value to y scale
        let [start, end] = this.yScale.domain();
        if(margin < start){
            start = margin;
        }
        if(margin > end){
            end = margin;
        }
        this.yScale.domain([start,end]);

        applyPaddingToScale(this.yScale, 0.01);
    }

    update(data, shutdowns, margin){
        // use cached values if needed
        data = data || this.data;
        shutdowns = shutdowns || this.shutdowns;
        margin = margin || this.margin;

        this.data = data;
        this.margin = margin;
        this.shutdowns = shutdowns;

        this.updateScales(data, margin);
        this.updateAxes();
        this.drawLines(data);
        this.drawMargin(margin);
        this.drawShutups(shutdowns);
    }

    resize(){
        this.calculateSize();
        this.svg.attr("width", this.w).attr("height", this.h);
        this.createScales();
        this.update(this.data, this.margin);
    }

    drawMargin(margin){
        let [left, right] = this.xScale.domain();

        let draw = d3.line()
            .x(d => this.xScale(d[0]))
            .y(d => this.yScale(d[1]));

        let marginLine = [[[left, margin], [right, margin]]];

        let line = this.visG.selectAll("path.margin")
            .data(marginLine);

        line.enter().append("path")
                .attr("class", "margin")
            .merge(line)
                .attr("d", d => draw(d));

        line.exit().remove();
    }

    drawLines(data){
        // draw lines
        let drawLine = d3.line()
            .x(d => this.xScale(d[0]))
            .defined(d => this.yScale(d[1]))
            .y(d => this.yScale(d[1]));

        let line = this.visG.selectAll("path.line-series")
            .data(data);

        line.enter().append("path")
                .attr("class", d => {
                    let classes = ["line-series"];
                    if(d.derived){
                        classes.push("derived");
                    }
                    return classes.join(" ");
                })
                .attr("data-id", d => d.metric)
            .merge(line)
                .attr("d", d => drawLine(d.data))
                .style("stroke", d => d.color);

        line.exit().remove();
    }

    drawShutups(shutdowns){

        let top = this.yScale.domain()[1],
            bottom = this.yScale.domain()[0];

        let drawShutdownLine = d3.line()
            .x(d => this.xScale(d[0]))
            .y(d => this.yScale(d[1]));

        let line = this.visG.selectAll("path.shutdown")
            .data(shutdowns.map(s => {
                return {
                    metric: s.metric,
                    dp: [[s.dp[0], s.dp[1]], [s.dp[0], bottom]]
                };
            }));

        line.enter().append("path")
                .attr("class", "shutdown")
            .merge(line)
                .attr("d", d => drawShutdownLine(d.dp));

        line.exit().remove();

        let circle = this.visG.selectAll("circle.shutdown_circle")
            .data(shutdowns);

        circle.enter().append("circle")
                .attr("class", "shutdown_circle")
            .merge(circle)
                .attr("cx", d => this.xScale(d.dp[0]))
                .attr("cy", d => this.yScale(d.dp[1]))
                .attr("r", 5);

        circle.exit().remove();
    }

    focusSeries(id){
        let line = this.visG.selectAll("path.line-series"),
            margin = this.visG.selectAll("path.margin"),
            shutdownLines = this.visG.selectAll("path.shutdown"),
            shutdownDots = this.visG.selectAll("circle.shutdown_circle");

        // return everything to normal
        if(id === null){
            line.classed("blurred", false);
            margin.classed("blurred", false);
            shutdownLines.classed("blurred", false);
            shutdownDots.classed("blurred", false);
            this.xAxis.tickValues(this.shutdowns.map(s => s.dp[0]));
        } else {

            // blur everyone but the specified id
            line.classed("blurred", d => d.metric !== id);
            // hide the margin line
            margin.classed("blurred", true);
            // hide all shutdowns that arent matching metric
            shutdownLines.classed("blurred", d => d.metric !== id);
            shutdownDots.classed("blurred", d => d.metric !== id);

            // update xAxis ticks to just these shutdowns
            this.xAxis.tickValues(this.shutdowns
                .filter(s => s.metric === id)
                .map(s => s.dp[0]));
        }

        // redraw x axis
        this.g.select("g.axis.axis-x")
            .call(this.xAxis)
        .selectAll(".tick text")
            .call(wrap, 0);
    }

    focusShutdown(metric, ts){
        let line = this.visG.selectAll("path.shutdown"),
            circle = this.visG.selectAll("circle.shutdown_circle"),
            id = metric + ts;

        if(metric === null){
            line.classed("blurred", false);
            circle.classed("blurred", false);
            this.xAxis.tickValues(this.shutdowns.map(s => s.dp[0]));
        } else {
            line.classed("blurred", d => (d.metric + d.dp[0][0]) !== id);
            circle.classed("blurred", d => (d.metric + d.dp[0]) !== id);
            // update xAxis ticks to just these shutdowns
            this.xAxis.tickValues(this.shutdowns
                .filter(s => (s.metric + s.dp[0]) === id)
                .map(s => s.dp[0]));
        }

        // redraw x axis
        this.g.select("g.axis.axis-x")
            .call(this.xAxis)
        .selectAll(".tick text")
            .call(wrap, 0);
    }
}

export function buildGraph(el){
    return new ThreshyGraph(el);
}
