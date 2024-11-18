#!/usr/bin/env python

import argparse
import sys

import compatlib


def get_parser():
    parser = argparse.ArgumentParser(
        description="Compatlib Python",
        formatter_class=argparse.RawTextHelpFormatter,
    )
    parser.add_argument(
        "--version",
        dest="version",
        help="show software version.",
        default=False,
        action="store_true",
    )

    subparsers = parser.add_subparsers(
        help="actions",
        title="actions",
        description="actions",
        dest="command",
    )

    # print version and exit
    subparsers.add_parser("version", description="show software version")

    # Write the install and configuration script
    analyze_recording = subparsers.add_parser(
        "analyze-recording",
        description="analyze one or more recordings from fs-record (in Go)",
        formatter_class=argparse.RawTextHelpFormatter,
    )
    analyze_recording.add_argument(
        "-p",
        "--prefix",
        help="Output prefix to replace (e.g., lammps-)",
    )
    analyze_recording.add_argument(
        "-s",
        "--suffix",
        help='Output suffix to replace (defaults to ".out")',
    )

    plot_recording = subparsers.add_parser(
        "plot-recording",
        description="analyze one or more recordings from fs-record (in Go)",
        formatter_class=argparse.RawTextHelpFormatter,
    )
    plot_recording.add_argument(
        "-n",
        help="Plot the top N paths",
        type=int,
        default=None,
    )

    run_models = subparsers.add_parser(
        "run-models",
        description="Build models for fs recordings",
        formatter_class=argparse.RawTextHelpFormatter,
    )

    for command in analyze_recording, plot_recording, run_models:
        command.add_argument(
            "-d",
            "--outdir",
            help="Output directory for images, events data frame, and results.",
        )
        command.add_argument("--name", help="Application name", default="LAMMPS")

    return parser


def run_compatlib():
    parser = get_parser()

    def help(return_code=0):
        version = compatlib.__version__

        print("\nCompatlib Python v%s" % version)
        parser.print_help()
        sys.exit(return_code)

    # If the user didn't provide any arguments, show the full help
    if len(sys.argv) == 1:
        help()

    # If an error occurs while parsing the arguments, the interpreter will exit with value 2
    args, extra = parser.parse_known_args()

    # Show the version and exit
    if args.command == "version" or args.version:
        print(compatlib.__version__)
        sys.exit(0)

    # retrieve subparser (with help) from parser
    helper = None
    subparsers_actions = [
        action for action in parser._actions if isinstance(action, argparse._SubParsersAction)
    ]
    for subparsers_action in subparsers_actions:
        for choice, subparser in subparsers_action.choices.items():
            if choice == args.command:
                helper = subparser
                break

    if args.command == "analyze-recording":
        from .analyze_recording import main
    if args.command == "plot-recording":
        from .plot_recording import main
    if args.command == "run-models":
        from .run_models import main

    # Pass on to the correct parser
    return_code = 0
    try:
        main(args=args, parser=parser, extra=extra, subparser=helper)
        sys.exit(return_code)
    except UnboundLocalError:
        return_code = 1
    help(return_code)


if __name__ == "__main__":
    run_compatlib()
