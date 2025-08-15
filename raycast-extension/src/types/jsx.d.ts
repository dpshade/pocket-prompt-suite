// Type fixes for Raycast API JSX component compatibility
declare namespace React {
  type ReactNode =
    | null
    | undefined
    | boolean
    | number
    | string
    | React.ReactElement
    | React.ReactNodeArray
    | React.ReactPortal;

  interface ReactNodeArray extends ReadonlyArray<ReactNode> {}
}

// @ts-ignore suppression for known JSX component type issues
declare global {
  namespace JSX {
    interface IntrinsicElements {
      [name: string]: any;
    }
  }
}
