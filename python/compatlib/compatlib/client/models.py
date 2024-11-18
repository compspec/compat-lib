import os
from itertools import cycle

import matplotlib.pylab as plt
from matplotlib.backends.backend_pdf import PdfPages

import compatlib.model as models
from compatlib.logger import logger
from compatlib.traces import TraceSet


def main(args, parser, extra, subparser):
    """
    The "extra" here is the list of events

    compatlib run-models $(find ../recording -name *.out)
    """
    # Extra events here should be one or more result event files to parse
    events = extra

    # An output directory is required
    if not args.outdir:
        logger.exit("Please specify an output directory with -d/--outdir")

    # A trace set is a collection of event files
    traceset = TraceSet(events)
    df = traceset.to_dataframe()

    # Define output files and paths
    image_outdir = os.path.join(args.outdir, "img")
    if not os.path.exists(image_outdir):
        os.makedirs(image_outdir)

    # Tell the user where the stuff is going!
    logger.info(f"Output will be saved to: {args.outdir}")
    logger.info(f"              🎨 Images: {image_outdir}")

    # Extra events here should be one or more result event files to parse
    df = traceset.to_dataframe()

    # Test 1: Simple Markov Model.
    # Let's generate a transition matrix based on unique paths
    # Let's do leave one out cross validation to use each as a test sample
    # And then just predict each path based on the previous and calculate
    # a total accuracy (correct / total) for the entire set.
    results = {"correct": 0, "wrong": 0}
    for train, test, _ in traceset.iter_loo():
        for key, value in models.build_and_test_markov(train, test).items():
            if key.startswith("transition"):
                continue
            results[key] += value

    print("Markov Model Results")
    print_results(results)

    # Let's compare to overall frequency - so ONE row that we do counts, and then
    # run the same procedure for. We could call this a 0-gram model :)
    frequency_results = models.build_and_test_frequency_model(traceset.samples)
    print("Frequency Results")
    print_results(frequency_results)

    # Test 2: Hidden Markov Models with Conditional Transition Times
    # We start with our same matrix of transition probabilities, but we also construct
    # a matrix of mean timeseries (Poisson distributed)
    # The results are residuals, so we can look at the error
    residuals = {}
    for train, test, left_out in traceset.iter_loo():
        # This takes the time in the state into account
        for path, new_residuals in models.build_and_test_markov_with_times(
            df, train, test, left_out
        ).items():
            if path not in residuals:
                residuals[path] = []
            residuals[path] += new_residuals

    plt.rcParams["figure.figsize"] = [7.00, 3.50]
    plt.rcParams["figure.autolayout"] = True

    # Make different colors!
    colors = cycle("bgrcmk")

    # Plot a histogram for each
    for path, residual_set in residuals.items():
        # This is important so each is a new figure
        plt.figure()
        plt.hist(residual_set, color=next(colors))
        plt.title(path)
    save_histogram_pdf(os.path.join(args.outdir, "residuals-for-paths-normalized.pdf"))
    plt.close()
    plt.clf()


# Saving functions
# Can be moved into separate module if/when shared


def save_histogram_pdf(save_path):
    """
    Save current plotting context to pdf with PdfPages.
    """
    # Get handles for all figures in plotting context
    fig_nums = plt.get_fignums()
    figs = [plt.figure(n) for n in fig_nums]

    # iterating over the numbers in list
    with PdfPages(save_path) as p:
        for fig in figs:
            fig.savefig(p, format="pdf")


def print_results(results):
    accuracy = results["correct"] / (results["correct"] + results["wrong"])
    print(f"  Leave one out correct: {results['correct']}")
    print(f"    Leave one out wrong: {results['wrong']}")
    print(f"          correct/total: {accuracy}")
