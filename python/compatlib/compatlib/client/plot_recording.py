import os

import matplotlib.pylab as plt
import pandas
import seaborn as sns

from compatlib.graph import Filesystem
from compatlib.logger import logger
from compatlib.traces import TraceSet


def main(args, parser, extra, subparser):
    """
    The "extra" here is the list of event files

    compatlib plot-recording $(find ../recording -name *.out)
    """
    # Extra events here should be one or more result event files to parse
    events = extra

    # An output directory is required
    if not args.outdir:
        logger.exit("Please specify an output directory with -d/--outdir")

    # A trace set is a collection of event files
    traceset = TraceSet(events)

    # Define output files and paths
    image_outdir = os.path.join(args.outdir, "img")
    if not os.path.exists(image_outdir):
        os.makedirs(image_outdir)

    # Tell the user where the stuff is going!
    logger.info(f"Output will be saved to: {args.outdir}")
    logger.info(f"              🎨 Images: {image_outdir}")

    # Get all counts so we can get the top N (if specified) across them
    counts = traceset.all_counts()
    if args.n is not None and len(counts) < args.n:
        args.n = len(counts) - 1
        shared_paths = list(counts)[: args.n]
    else:
        shared_paths = list(counts)

    # We will make a nice heatmap of files by recorded opens
    df = pandas.DataFrame(0, columns=shared_paths, index=traceset.files)

    for path, trace in traceset.as_counts(fullpath=True).items():
        single_trace = {p: 0 for p in shared_paths}
        for k, v in trace.items():
            # Don't add any not in the shared set
            if k not in single_trace:
                continue
            single_trace[k] = v
        df.loc[path] = single_trace

    plt.figure(figsize=(20, 20))
    df.index = [os.path.basename(x) for x in df.index]
    sns.clustermap(df, mask=(df == 0.0), cmap="tab20b")

    if args.n is not None:
        save_path = f"{args.name}-top-{args.n}-recorded-paths.png"
        title = f"{args.name} Top (N={args.n}) Recorded Paths"
        trie_title = f"Filesystem Recording Trie for {args.name} Top {args.n} Recorded Paths"
        trie_save = f"{args.name}-top-{args.n}-recorded-paths-trie.png"
    else:
        save_path = f"{args.name}-recorded-paths.png"
        title = f"{args.name} Top Recorded Paths"
        trie_title = f"Filesystem Recording Trie for {args.name} Top Recorded Paths"
        trie_save = f"{args.name}-top-recorded-paths-trie.png"

    # Save all the things!
    plot_path = os.path.join(image_outdir, save_path)
    plt.title(title.rjust(70), fontsize=10)
    plt.tight_layout()
    plt.savefig(plot_path)
    plt.close()

    # Next, let's create a plot across binaries of a Trie
    fs = Filesystem()
    for i, (path, count) in enumerate(counts.items()):
        # Only plot top N requested by user
        if args.n is not None and i > args.n:
            break
        fs.insert(path, count=count)

    # Generate graph. This adds to matplotlib context
    fs.get_graph(title=trie_title)
    plot_path = os.path.join(image_outdir, trie_save)
    plt.savefig(plot_path)
